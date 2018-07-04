package main

import (
	"flag"
	"fmt"
	"gopkg.in/ini.v1"
	"log"
	"os/exec"
	"strings"
	"time"
)

func checkSliceValue(slice []string, target string) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

func main() {

	now := time.Now()
	today := fmt.Sprintf("%d%d%d-%d%d-%d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

	repo := flag.String("repo", "", "Repository name (Eg. example.com)")
	path := flag.String("path", "", "Path on the repository (Eg. /ManagerRepo/trunk")
	user := flag.String("user", "", "User to set or add (Eg. user123)")
	perm := flag.String("perm", "r", "User permission (Eg. read-write)")

	flag.Parse()

	reposBaseDir := "/var/svn-repos"
	fileRepoPath := fmt.Sprintf("file://%s/%s", reposBaseDir, *repo)
	cmd := exec.Command("/usr/bin/svn", "info", fileRepoPath)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Repository might not be existing on the ", reposBaseDir)
		log.Fatal(err)
	}
	permission := *perm

	if permission == "r-w" {
		permission = "read-write"
	}

	if permission == "r" || permission == "read" {
		permission = "read-only"
	}

	configFile := fmt.Sprintf("%s/%s/hooks/commit-access-control.cfg", reposBaseDir, *repo)

	//fmt.Println(configFile)

	cfg, err := ini.Load(configFile)
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Printf("repo: %s, path: %s, user: %s, perm: %s", *repo, *path, *user, *perm)
	/*
		for _, section := range cfg.Sections() {
			keyshash := section.KeysHash()
			fmt.Println(keyshash)
			for k, v := range keyshash {
				fmt.Printf("key: %s,value: %s\n", k, v)
			}
		}
	*/
	targetSection := fmt.Sprintf("Make repos%s %s", *path, permission)
	pathPattern := fmt.Sprintf("^%s/.*$", *path)
	ts, err := cfg.GetSection(targetSection)
	configFileBackup := fmt.Sprintf("%s-%s", configFile, today)
	cfg.SaveTo(configFileBackup)

	if err != nil {
		fmt.Println("Section \"", targetSection, "\" does not exist. Creating...")
		cfg.Section(targetSection).Key("match").SetValue(pathPattern)
		cfg.Section(targetSection).Key("users").SetValue(*user)
		cfg.Section(targetSection).Key("access").SetValue(permission)
		cfg.SaveTo(configFile)
		fmt.Println("Section has already been created.")
	} else {
		fmt.Println("There is already an existing section \"", targetSection, "\".")
		cfg.Section(targetSection).Key("match").SetValue(pathPattern)
		users := ts.Key("users")
		usersSlice := strings.Split(users.String(), " ")
		if checkSliceValue(usersSlice, *user) != true {
			fmt.Println("Adding user ", *user)
			updatedUsers := fmt.Sprintf("%s %s", users.String(), *user)
			cfg.Section(targetSection).Key("users").SetValue(updatedUsers)
			cfg.SaveTo(configFile)
		} else {
			fmt.Println("User ", *user, " already exists.")
		}
	}
}
