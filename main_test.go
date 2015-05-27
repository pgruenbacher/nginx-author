package main

import (
	"fmt"
	"testing"
)

var group1 = Group{
	filename: ".htpasswdTestNewGroup",
}
var user1 = User{
	name:     "test1",
	Password: "testpasswd",
}
var user2 = User{
	name:     "test2",
	Password: "testpasswd2",
}
var userss = []*User{
	&user1,
	&user2,
}

func TestNewGroup(t *testing.T) {
	newGroup(&group1, userss)
	deleteGroup(&group1)
}

func TestReadConfig(t *testing.T) {
	config := readConfig("testConfig.toml")
	if err := organizeConfig(&config); err != nil {
		t.Error(err)
	}
	fmt.Println(config)
	writeLocations("sites-enabled/default", config.Groups)
	writeFiles(config.Groups)
}
