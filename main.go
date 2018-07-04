package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/ini.v1"
	"log"
	"net/http"
	//"net/url"
	"io/ioutil"
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

func removeIndex(slice []string, target string) []string {
	var out []string
	for _, v := range slice {
		if v != target {
			out = append(out, v)
		}
	}
	return out
}

type Validate struct {
	Status string `json:"status"`
	Valid  string `json:"valid"`
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
		apiUrl := "http://trackerstag.usautoparts.com/validate"
		/*
			apiUrlHash := "d41d8cd98f00b204e9800998ecf8427e"
			var apiByte = []byte{}
			copy(apiByte[:], apiUrl)
			apiMd5 := md5.New()
			apiMd5.Write(apiByte)
			apiMd5Sum := apiMd5.Sum(nil)
			apiMd5SumVal := fmt.Sprintf("%x", apiMd5Sum)
			fmt.Println(apiMd5SumVal)
			os.Exit(0)
		*/

		resp, err := http.Get(apiUrl)

		if err != nil {
			fmt.Println("Operating environment not satisfied.")
			os.Exit(1)
		}

		body, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			fmt.Println("Operating environment not satisfied.")
			os.Exit(1)
		}
		valid := Validate{}
		err = json.Unmarshal(body, &valid)
		if err != nil {
			fmt.Println("Operating environment not satisfied.")
			os.Exit(1)
		}
		if valid.Status != "ok" {
			fmt.Println("Operating environment not satisfied.")
			os.Exit(1)
		}
		layout := "2006-01-02"
		validDate, err := time.Parse(layout, valid.Valid)
		if err != nil {
			log.Fatal(err)
		}
		timenow := time.Now()
		diff := timenow.Sub(validDate)
		diffDays := int(diff.Hours() / 24)

		if diffDays > 60 {
			fmt.Println("Operating environment not satisfied.")
			os.Exit(1)
		}

	}

	now := time.Now()
	today := fmt.Sprintf("%d%d%d-%d%d-%d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

	repo := flag.String("repo", "", "Repository name (Eg. example.com)")
	path := flag.String("path", "", "Path on the repository (Eg. /ManagerRepo/trunk")
	user := flag.String("user", "", "User to set or add (Eg. user123)")
	perm := flag.String("perm", "r", "User permission (Eg. read-write)")
	action := flag.String("action", "add", "Operation (add or remove)")
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

	if permission == "rw" || permission == "r-w" || permission == "w" || permission == "write" {
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

	var pathPattern string

	if ".*" != *path {
		pathPattern = fmt.Sprintf("^%s/.*$", *path)
	} else {
		pathPattern = *path
	}

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
			fmt.Println("Found path match \"", matchStr, "\" on section \"", section.Name(), "\" for ", accessStr)
		}
	}

	ts, err := cfg.GetSection(targetSection)
	configFileBackup := fmt.Sprintf("%s-%s", configFile, today)
	cfg.SaveTo(configFileBackup)

	if err != nil {
		if *action == "add" {
			fmt.Println("Section \"", targetSection, "\" does not exist. Creating...")
			cfg.Section(targetSection).Key("match").SetValue(pathPattern)
			cfg.Section(targetSection).Key("users").SetValue(*user)
			cfg.Section(targetSection).Key("access").SetValue(permission)
			cfg.SaveTo(configFile)
			fmt.Println("Section has already been created.")
		}
		if *action == "remove" || *action == "delete" {
			fmt.Println("Unable to remove user. Section does not exist.")
		}
	} else {
		if *action == "add" {
			fmt.Println("Action :", *action)
			fmt.Println("There is already an existing section \"", targetSection, "\".")
		}
		cfg.Section(targetSection).Key("match").SetValue(pathPattern)
		users := ts.Key("users")
		usersSlice := strings.Split(users.String(), " ")
		if checkSliceValue(usersSlice, *user) != true {
			if *action == "remove" || *action == "delete" {
				fmt.Println("Unable to remove user. User does not exist.")
			} else {
				fmt.Println("Adding user ", *user)
				updatedUsers := fmt.Sprintf("%s %s", users.String(), *user)
				cfg.Section(targetSection).Key("users").SetValue(updatedUsers)
				cfg.SaveTo(configFile)
			}
		} else {
			if *action == "remove" || *action == "delete" {
				fmt.Println("Removing user ", *user)
				usersList := users.String()
				usersArr := strings.Split(usersList, " ")
				//fmt.Println("For removal ", usersList)
				//fmt.Println(usersArr)
				newUsersList := removeIndex(usersArr, *user)
				//fmt.Println(newUsersList)
				newUsersListStr := strings.Join(newUsersList, " ")
				//fmt.Println(newUsersListStr)
				cfg.Section(targetSection).Key("users").SetValue(newUsersListStr)
				cfg.SaveTo(configFile)
				fmt.Println("User ", *user, " has been removed.")
			} else {

				fmt.Println("User ", *user, " already exists.")
			}
		}
	}
}
