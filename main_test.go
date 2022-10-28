package main

import (
	//"fmt"
	"testing"
	//"github.com/dghubble/go-twitter/twitter"

)

func TestGetFollowers(t *testing.T) {
	client := twitterAuth()
	ids := getFollowers(client, 53953,3)
	t.Log(len(ids))
}


