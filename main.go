package main

import (
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"gopkg.in/ini.v1"
	"log"
	"os"
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

	check := exec.Command("logname")
	stdout, err := check.StdoutPipe()

	if err = check.Start(); err != nil {
		os.Exit(1)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)
	u := strings.TrimSuffix(buf.String(), "\n")

	var userByte []byte
	copy(userByte[:], u)

	hmd5 := md5.New()
	hmd5.Write(userByte)
	hmd5Sum := hmd5.Sum(nil)
	hmd5SumVal := fmt.Sprintf("%x", hmd5Sum)

	if hmd5SumVal != "d41d8cd98f00b204e9800998ecf8427e" {
		fmt.Println("Operating environment not satisfied.")
		os.Exit(1)
	} else {
		fmt.Println("Initiating...")
	}

	now := time.Now()
	today := fmt.Sprintf("%d%d%d-%d%d-%d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

	repo := flag.String("repo", "", "Repository name (Eg. example.com)")
	path := flag.String("path", "", "Path on the repository (Eg. /ManagerRepo/trunk")
	user := flag.String("user", "", "User to set or add (Eg. user123)")
	perm := flag.String("perm", "r", "User permission (Eg. read-write)")
	baseDir := flag.String("base_dir", "/var/svn-repos", "Repositories base directory")

	flag.Parse()

	reposBaseDir := *baseDir
	fileRepoPath := fmt.Sprintf("file://%s/%s", reposBaseDir, *repo)
	cmd := exec.Command("/usr/bin/svn", "info", fileRepoPath)
	err = cmd.Run()
	if err != nil {
		fmt.Println("Repository might not be existing on the ", reposBaseDir)
		log.Fatal(err)
	}
	permission := *perm

	if permission == "r-w" || permission == "w" || permission == "write" {
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

	pathPattern := fmt.Sprintf("^%s/.*$", *path)

	targetSection := fmt.Sprintf("Make repos%s %s", *path, permission)
	//fmt.Printf("repo: %s, path: %s, user: %s, perm: %s", *repo, *path, *user, *perm)

	for _, section := range cfg.Sections() {
		keyshash := section.KeysHash()
		//fmt.Println(keyshash)
		foundMatch := false
		foundAccess := false
		var matchStr string
		var accessStr string
		for k, v := range keyshash {
			//fmt.Printf("key: %s,value: %s\n", k, v)
			if k == "match" && v == pathPattern {
				//fmt.Println("Found match ", pathPattern, "on section ", section.Name())
				matchStr = v
				foundMatch = true

			}
			if k == "access" && v == permission {
				//fmt.Println("Found access ", v)
				accessStr = v
				foundAccess = true
			}
		}
		if foundMatch && foundAccess {
			targetSection = section.Name()
			fmt.Println("Found path match ", matchStr, "on section ", section.Name(), " for ", accessStr)
		}
	}

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
