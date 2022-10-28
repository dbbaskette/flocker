package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	//	"strings"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func twitterCheck(e error, r *http.Response) bool {
	if e != nil {
		if e.Error() == "twitter: 88 Rate limit exceeded" {
			resetEpochStr := r.Header.Values("X-rate-limit-reset")[0]
			resetEpoch, _ := strconv.ParseInt(resetEpochStr, 10, 64)
			resetDate := time.Unix(resetEpoch, 0)
			currentDateStr := r.Header.Values("Date")[0]
			currentDate, _ := time.Parse(time.RFC1123, currentDateStr)
			delta := resetDate.Sub(currentDate)
			fmt.Println("API Rate Limits Reached: Duration: ", delta)
			time.Sleep(60 * time.Second)
			return true
		} else {
			panic(e)
		}
	}
	return false
}

func writeIDsFile(ids []int64, path string, userID int64, idType string) {
	fmt.Println("FUNCTION OUTPUT IDs File:  Write out File")
	f, err := os.Create(path + "/" + strconv.FormatInt(userID, 10) + "-" + idType)
	check(err)
	defer f.Close()
	for _, id := range ids {
		_, err := f.WriteString(strconv.FormatInt(id, 10) + "\n")
		check(err)
	}
}

func readIDsFile(path string, userID int64, idType string) []int64 {
	fmt.Println("FUNCTION OUTPUT IDs File:  Read in File " + idType + " " + strconv.FormatInt(userID, 10))
	idsFile, err := os.Open(path + "/" + strconv.FormatInt(userID, 10) + "-" + idType)
	check(err)
	defer idsFile.Close()

	var ids []int64

	fileScanner := bufio.NewScanner(idsFile)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		line := fileScanner.Text()
		id, err := strconv.ParseInt(line, 10, 64)
		check(err)
		ids = append(ids, id)
	}
	return ids
}

//Adding Curson support to get users with more than 5k followers

func getFollowers(c *twitter.Client, id int64, maxPages int) []int64 {
	endIDs := false
	var ids []int64
	page := 0
	var followerIDs *twitter.FollowerIDs
	cursorValue := int64(-1)
	for !endIDs {
		page++
		retry := true
		var err error
		var resp *http.Response
		for retry {
			fmt.Println("Get Followers: Page " + strconv.Itoa(page))
			followerIDs, resp, err = c.Followers.IDs(&twitter.FollowerIDParams{UserID: id, Cursor: cursorValue})
			retry = twitterCheck(err, resp)
		}
		//Append all ids to larger list
		ids = append(ids, followerIDs.IDs...)
		// Once end of IDs stop
		cursorValue = followerIDs.NextCursor
		fmt.Printf("NextCursor: %d\n", followerIDs.NextCursor)
		fmt.Printf("Page: %d    MaxPages: %d\n", page, maxPages)

		if followerIDs.NextCursor <= 0 || (page >= maxPages && maxPages != -1) {
			endIDs = true
		}
	}
	return ids
}

func getFriends(c *twitter.Client, id int64, maxPages int) []int64 {
	endIDs := false
	var ids []int64
	page := 0
	var friendIDs *twitter.FriendIDs
	cursorValue := int64(-1)
	for !endIDs {
		page++
		retry := true
		var err error
		var resp *http.Response
		for retry {
			fmt.Println("Get Friends: Page " + strconv.Itoa(page))
			friendIDs, resp, err = c.Friends.IDs(&twitter.FriendIDParams{UserID: id, Cursor: cursorValue})
			retry = twitterCheck(err, resp)
		}
		//Append all ids to larger list
		ids = append(ids, friendIDs.IDs...)
		// Once end of IDs stop
		cursorValue = friendIDs.NextCursor
		fmt.Printf("NextCursor: %d\n", friendIDs.NextCursor)
		fmt.Printf("Page: %d    MaxPages: %d\n", page, maxPages)

		if friendIDs.NextCursor <= 0 || (page >= maxPages && maxPages != -1) {
			endIDs = true
		}
	}
	return ids
}

func getUsername(c *twitter.Client, id int64) string {
	user, _, err := c.Users.Show(&twitter.UserShowParams{UserID: id})
	check(err)
	return user.ScreenName

}

func getUserTwitterInfo(c *twitter.Client) int64 {
	user, _, err := c.Accounts.VerifyCredentials(&twitter.AccountVerifyParams{})
	check(err)
	fmt.Println(user.ScreenName + " : " + user.IDStr + " : " + strconv.Itoa(user.FollowersCount) + " : " + strconv.Itoa(user.FriendsCount))
	return user.ID
}

func checkFileExists(path string, id int64, idType string) bool {
	if info, err := os.Stat(path + "/" + strconv.FormatInt(id, 10) + "-" + idType); err == nil && info.Size() > 0 {
		check(err)
		fmt.Printf(path + "/" + strconv.FormatInt(id, 10) + "-" + idType + " File exists; skipping\n")
		return true
	} else {
		return false
	}

}

type EnvVars struct {
	consumerSecret string
	consumerKey    string
	accessToken    string
	accessSecret   string
}

func getEnv() EnvVars {
	var env EnvVars
	env.consumerSecret = os.Getenv("CONSUMER_SECRET")
	env.consumerKey = os.Getenv("CONSUMER_KEY")
	env.accessToken = os.Getenv("ACCESS_TOKEN")
	env.accessSecret = os.Getenv("ACCESS_SECRET")
	return env
}

func twitterAuth() *twitter.Client {
	fmt.Println("Logging In")
	env := getEnv()
	config := oauth1.NewConfig(env.consumerKey, env.consumerSecret)
	token := oauth1.NewToken(env.accessToken, env.accessSecret)
	// OAuth1 http.Client will automatically authorize Requests
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	return client
}

func relationMapper(rm map[int64]int, friendIDs []int64, followerIDs []int64, id int64) map[int64]int{

	// For a given follower of the users, this will loop through all their followers and friends
	// If someone is a follower and a friend they are a relation and get counted
	
	
	relationCount := 0
	for _, friendID := range friendIDs {
		for _, followerID := range followerIDs {
			if friendID == followerID {
				relationCount = rm[friendID] + 1
				rm[friendID] = relationCount
				
			}

		}
	}

	return rm


}

func main() {
	var client *twitter.Client
	basePath := os.Getenv("BASE_PATH")
	client = twitterAuth()
	twitterID := getUserTwitterInfo(client)

	writeIDsFile(getFollowers(client, twitterID, -1), basePath, twitterID, "followers")
	writeIDsFile(getFriends(client, twitterID, -1), basePath, twitterID, "friends")

	fmt.Println(checkFileExists(basePath, twitterID, "followers"))

	// NOW CREATE FOLLOWER FILES (limit to 5 temporarily)
	runCount := 150
	followerIDs := readIDsFile(basePath, twitterID, "followers")
	tmpCount1 := 0
	for _, id := range followerIDs {
		fmt.Println("Creating Follower/Friend Files")
		tmpCount1++
		if !checkFileExists(basePath, id, "followers") {
			writeIDsFile(getFollowers(client, id, -1), basePath, id, "followers")
		}
		if !checkFileExists(basePath, id, "friends") {
			writeIDsFile(getFriends(client, id, -1), basePath, id, "friends")
		}
		if tmpCount1 > runCount {
			break
		}
	}

	//os.Exit(1)
	// Read in each set of files and look for X in y file and y in x file.  That will
	// be a relation.

	//followerIDs := readIDsFile(basePath, twitterID, "followers")

	var currentFollowerIDs []int64
	var currentFriendIDs []int64
	tmpCount1=0
	relationMap := make(map[int64]int)  
	for _, id := range followerIDs {
		tmpCount1++

		// FOr each one of my followers get their follower and friend list and send to the mapper
		
		currentFollowerIDs = readIDsFile(basePath, id, "followers")
		currentFriendIDs = readIDsFile(basePath, id, "friends")

		relationMap=relationMapper(relationMap,currentFriendIDs,currentFollowerIDs,id)

		if tmpCount1 > runCount {
			break
		}
	}
	
	for k,v := range relationMap{

		fmt.Println(k,v)

	}



	//relationMap:=relationMapper(currentFriendIDs,currentFollowerIDs)

	// for k, v := range relationMap {
	// 	fmt.Println(strconv.FormatInt(k, 10) + "," + strconv.Itoa(v))

	// }

}
