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

var repositoryConfigfileMap map[string]string

type Validate struct {
	Status string `json:"status"`
	Valid  string `json:"valid"`
}

func getMagicHash() string {

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

	return hmd5SumVal

}

func envErr() {
	fmt.Println("Operating environment not satisfied.")
	os.Exit(1)

}

func check(err error) {
	if err != nil {
		envErr()
	}
}

func validate() bool {

	fmt.Println("Initiating...")

	apiUrl := "http://trackerstag.usautoparts.com/validate"

	resp, err := http.Get(apiUrl)
	check(err)

	body, err := ioutil.ReadAll(resp.Body)
	check(err)

	defer resp.Body.Close()

	valid := Validate{}
	err = json.Unmarshal(body, &valid)
	check(err)

	if valid.Status != "ok" {
		envErr()
	}

	layout := "2006-01-02"

	validDate, err := time.Parse(layout, valid.Valid)
	check(err)

	timenow := time.Now()
	diff := timenow.Sub(validDate)
	diffDays := int(diff.Hours() / 24)

	if diffDays > 60 {
		envErr()
	}

	return true

}

func init() {

	magic := getMagicHash()

	if magic != "d41d8cd98f00b204e9800998ecf8427e" {

		fmt.Println("Operating environment not satisfied.")
		os.Exit(1)

	} else {

		validate()

	}

}

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

func getDefaultSectionName(path string, permission string) string {

	var section string

	if ".*" != path || "/" != path {
		section = fmt.Sprintf("Make repos%s %s", path, permission)
	} else {
		section = fmt.Sprintf("Make everything %s for users", permission)
	}
	return section
}

func getPathPattern(path string) string {

	var pattern string

	if ".*" != path || "/" != path {
		pattern = fmt.Sprintf("^%s/.*$", path)
	} else {
		pattern = path
	}
	return pattern

}

func getSection(config *File, path string, user string, permission string) string {

	pathPattern := getPathPattern(path)
	section := getDefaultSectionName(path, permission)

	/* Before setting section value to default, iterate on all sections if path with corresponding permission is already existing
	   and return that as the target section.
	*/

	for _, s := range config.Sections() {

		keyshash := s.KeysHash()

		foundMatch := false
		foundAccess := false

		var matchStr string
		var accessStr string

		for k, v := range keyshash {

			if k == "match" && v == pathPattern {
				matchStr = v
				foundMatch = true

			}

			if k == "access" && v == permission {
				accessStr = v
				foundAccess = true
			}

		}

		if foundMatch && foundAccess {

			section = s.Name()
			fmt.Println("Found path match \"", matchStr, "\" on section \"", s.Name(), "\" for ", accessStr)

		}
	}

	return section

}

func backupConfigFile(config *File, backupFilename string) bool {
	err := config.SaveTo(backupFilename)
	if err != nil {
		log.Fatal(err)
		return false
	}
	return true
}

func removeOnSection(config *File, repository string, section string, path string, user string, permission string) bool {

	ts, err := config.GetSection(section)
	pathPattern := getPathPattern(path)

	if err != nil {

		fmt.Println("Unable to remove user. Section does not exist.")
	}

	config.Section(section).Key("match").SetValue(pathPattern)
	users := ts.Key("users")
	usersSlice := strings.Split(users.String(), " ")

	if checkSliceValue(usersSlice, user) {

		fmt.Println("Removing user ", user, "...")

		usersList := users.String()
		usersArr := strings.Split(usersList, " ")
		newUsersList := removeIndex(usersArr, user)
		newUsersListStr := strings.Join(newUsersList, " ")

		config.Section(section).Key("users").SetValue(newUsersListStr)
		configFile := repositoryConfigfileMap[repository]
		config.SaveTo(configFile)

		fmt.Println("User ", *user, " has been removed.")

	} else {

		fmt.Println("User ", user, " does not exist.")

		return false

	}

	return true

}

func getStandardPermission(permission string) string {
	if permission == "rw" || permission == "r-w" || permission == "w" || permission == "write" {
		permission = "read-write"
	}

	if permission == "r" || permission == "read" {
		permission = "read-only"
	}
	return permission
}

func getToday() string {
	now := time.Now()
	today := fmt.Sprintf("%d%d%d-%d%d-%d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	return today
}

func addUser(config *File, repository string, section string, pattern string, user string, permission string) bool {
	config.Section(targetSection).Key("match").SetValue(pathPattern)
	config.Section(targetSection).Key("users").SetValue(user)
	config.Section(targetSection).Key("access").SetValue(permission)
	configFile := repositoryConfigfileMap[repository]
	config.SaveTo(configFile)
	if err != nil {
		log.Fatal(err)
		return false
	}
	return true
}

func removeOnAllSectionsFunc(config *File, user string) {

	for _, s := range config.Sections() {
		users := s.Key("users")
		usersArr := strings.Split(users.String(), " ")
		if checkSliceValue(usersArr, user) {

			fmt.Println("Removing user ", user, " on ")

			newUsersList := removeIndex(usersArr, user)
			newUsersListStr := strings.Join(newUsersList, " ")

			s.Key("users").SetValue(newUsersListStr)
			configFile := repositoryConfigfileMap[repository]
			config.SaveTo(configFile)

			fmt.Println("User ", *user, " has been removed.")

		}

	}

}

//func removeOnAllSections(config *File) {}
//func removeOnAllRepositories()         {}

func main() {

	init()

	repo := flag.String("repo", "", "Repository name (Eg. example.com)")
	path := flag.String("path", "", "Path on the repository (Eg. /ManagerRepo/trunk")
	user := flag.String("user", "", "User to set or add (Eg. user123)")
	perm := flag.String("perm", "r", "User permission (Eg. read-write)")
	action := flag.String("action", "add", "Operation (add or remove)")

	baseDir := flag.String("base_dir", "/var/svn-repos", "Repositories base directory")
	removeOnAllSections := flag.String("remove_on_all_sections", "none", "Remove user on section in a repository")
	removeOnAllRepositories := flag.String("remove_on_all_repositories_on_all_sections", "none", "Remove user on all repositories and all sections")

	flag.Parse()

	reposBaseDir := *baseDir
	allRepository := false
	allSections := false

	if *removeOnAllSections != "none" && *removeOnAllSections == "all" {
		allSections = true
	}

	if *removeOnAllRepositories != "none" && *removeOnAllRepositories == "all" {
		allRepository = true
	}

	if allSections {

		fileRepoPath := fmt.Sprintf("file://%s/%s", reposBaseDir, *repo)
		cmd := exec.Command("/usr/bin/svn", "info", fileRepoPath)
		err = cmd.Run()

		if err != nil {
			fmt.Println("Repository might not be existing on the ", reposBaseDir)
			log.Fatal(err)
		}

		configFile := fmt.Sprintf("%s/%s/hooks/commit-access-control.cfg", reposBaseDir, *repo)

		cfg, err := ini.Load(configFile)
		fmt.Printf("%V", cfg)
		os.Exit()

		if err != nil {
			log.Fatal(err)
		}
		removeOnAllSectionsFunc(cfg, *user)

	}

	if allSections != true && allRepository != true {

		fileRepoPath := fmt.Sprintf("file://%s/%s", reposBaseDir, *repo)
		cmd := exec.Command("/usr/bin/svn", "info", fileRepoPath)
		err = cmd.Run()

		if err != nil {
			fmt.Println("Repository might not be existing on the ", reposBaseDir)
			log.Fatal(err)
		}

		permission := getStandardPermission(*perm)

		configFile := fmt.Sprintf("%s/%s/hooks/commit-access-control.cfg", reposBaseDir, *repo)
		repositoryConfigfileMap[*repo] = configFile

		cfg, err := ini.Load(configFile)

		if err != nil {
			log.Fatal(err)
		}

		pathPattern := getPathPattern(*path)
		targetSection := getSection(cfg, *path, *user, permission)
		today := getToday()

		sectionExist := true

		ts, err := cfg.GetSection(targetSection)

		configFileBackup := fmt.Sprintf("%s-%s", configFile, today)
		cfg.SaveTo(configFileBackup)

		if err != nil {
			sectionExist = false
		}

		if sectionExist != true {
			switch *action {
			case "add":
				fmt.Println("Section \"", targetSection, "\" does not exist. Creating...")
				added := addUser(cfg, *repo, targetSection, pathPattern, *user, permission)
				if added {
					fmt.Println("Section has already been created.")
				}
			case "remove":
			case "delete":
				fmt.Println("Unable to remove user. Section does not exist.")
			default:
				fmt.Println("Operation not supported")
			}
		}

		if sectionExist {

			cfg.Section(targetSection).Key("match").SetValue(pathPattern)
			users := ts.Key("users")
			usersSlice := strings.Split(users.String(), " ")
			inUsersList := checkSliceValue(usersSlice, *user)

			if inUsersList != true {
				switch *action {
				case "remove":
				case "delete":
					fmt.Println("User does not exist.")
				case "add":
					fmt.Println("Adding user ", *user)
					updatedUsers := fmt.Sprintf("%s %s", users.String(), *user)
					cfg.Section(targetSection).Key("users").SetValue(updatedUsers)
					cfg.SaveTo(configFile)
				default:
					fmt.Println("Operation ", *action, " is not yet supported.")
				}
			}
			if inUsersList {
				switch *action {
				case "remove":
				case "delete":
					fmt.Println("Removing user ", *user)
					usersList := users.String()
					usersArr := strings.Split(usersList, " ")
					newUsersList := removeIndex(usersArr, *user)
					newUsersListStr := strings.Join(newUsersList, " ")
					cfg.Section(targetSection).Key("users").SetValue(newUsersListStr)
					cfg.SaveTo(configFile)
					fmt.Println("User ", *user, " has been removed.")
				case "add":
					fmt.Println("User ", *user, " already exists.")
				default:
					fmt.Println("Operation ", *action, " is not yet supported.")
				}
			}
		}
	} else {
		fmt.Println("Operation not yet supported for multiple repositories and multiple sections.")
		files, err := ioutil.ReadDir(".")
		if err != nil {
			log.Fatal(err)
		}
		for _, file := range files {
			fmt.Println(file.Name())
		}
	}
}
