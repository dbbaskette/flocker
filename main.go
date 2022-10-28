package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"

	//"net/http"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func twitterCheck(e error) bool {
	if e != nil {
		if e.Error() == "twitter: 88 Rate limit exceeded" {
			fmt.Println("You have hit the Twitter API Rate Limits. Pausing for 1 minute.")
			time.Sleep(60 * time.Second)
			return false
		} else {
			panic(e)
		}
	}
	return true
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
	fmt.Println("FUNCTION OUTPUT IDs File:  Read in File "+idType+" "+strconv.FormatInt(userID, 10))
	idsFile, err := os.Open(path + "/" + strconv.FormatInt(userID, 10) + "-" + idType )
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

func getFollowers(c *twitter.Client, id int64) []int64 {
	retry := true
	var followerIDs *twitter.FollowerIDs
	var err error
	for ok := true; ok; ok = !retry {
		followerIDs, _, err = c.Followers.IDs(&twitter.FollowerIDParams{UserID: id})
		retry = twitterCheck(err)
	}
	return followerIDs.IDs
}

func getFriends(c *twitter.Client, id int64) []int64 {
	retry := true
	var friendIDs *twitter.FriendIDs
	var err error
	for ok := true; ok; ok = !retry {
		friendIDs, _, err = c.Friends.IDs(&twitter.FriendIDParams{UserID: id})
		retry = twitterCheck(err)
	}
	return friendIDs.IDs
}

func getUsername(c *twitter.Client, id int64) string{
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

func main() {

	consumerSecret := os.Getenv("CONSUMER_SECRET")
	consumerKey := os.Getenv("CONSUMER_KEY")
	accessToken := os.Getenv("ACCESS_TOKEN")
	accessSecret := os.Getenv("ACCESS_SECRET")
	basePath := os.Getenv("BASE_PATH")
	//twitterID := os.Getenv("TWITTER_ID")
	fmt.Println("Logging In")

	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	// OAuth1 http.Client will automatically authorize Requests
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	twitterID := getUserTwitterInfo(client)

	writeIDsFile(getFollowers(client, twitterID), basePath, twitterID, "followers")
	writeIDsFile(getFriends(client, twitterID), basePath, twitterID, "friends")

	fmt.Println(checkFileExists(basePath, twitterID, "followers"))

	// NOW CREATE FOLLOWER FILES (limit to 5 temporarily)

	followerIDs := readIDsFile(basePath, twitterID, "followers")
	tmpCount1 := 0
	for _, id := range followerIDs {
		fmt.Println("Creating Follower/Friend Files")
		tmpCount1++
		if !checkFileExists(basePath, id, "followers") {
			writeIDsFile(getFollowers(client, id), basePath, id, "followers")
		}
		if !checkFileExists(basePath, id, "friends") {
			writeIDsFile(getFriends(client, id), basePath, id, "friends")
		}
		if tmpCount1 > 20 {
			break
		}
	}

	// Read in each set of files and look for X in y file and y in x file.  That will
	// be a relation.

	//followerIDs := readIDsFile(basePath, twitterID, "followers")
	
	var currentFollowerIDs []int64
	var currentFriendIDs []int64
	tmpCount1 = 0
	for _, id := range followerIDs {
		tmpCount1++
		fmt.Println(getUsername(client,id)) 
		currentFollowerIDs = readIDsFile(basePath, id, "followers")
		currentFriendIDs = readIDsFile(basePath, id, "friends")
		if tmpCount1 > 5 {
			break
		}
	}


// Now, Check all friendIDs and check for that ID in the followers list. Store those as relations
relationMap := make(map[int64]int)
relationCount:=0
for _,currentFriendID := range currentFriendIDs{
	for _, currentFollowerID := range currentFollowerIDs{
		if currentFriendID==currentFollowerID{
			relationCount=relationMap[currentFriendID]+1
			relationMap[currentFriendID]=relationCount
		}

	}
}



for k,v := range relationMap{
	//fmt.Println(getUsername(client, k)+" : "+strconv.Itoa(v))
	fmt.Println(getUsername(client, k)+" : "+strconv.FormatInt(k,10)+" : "+strconv.Itoa(v))

}

	

}
