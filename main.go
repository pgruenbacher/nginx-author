package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

type User struct {
	name     string
	Password string
	Group    []string
}

type Group struct {
	name     string
	filename string
	Users    []*User
	Location string
}

const (
	Global string = "ALL"
	start  string = "START_AUTH_LOCATIONS"
	end    string = "END"
)

type Groups map[string]*Group

type Users map[string]*User

var configPath = "authConfig.toml"
var locationsPath = "sites-enabled/default"
var passwordPath = "auth-test/.htpasswd"

func main() {
	var AuthCmd = &cobra.Command{
		Use:   "auth",
		Short: "auth helps organize User authentication and Groups",
		Long: `
A command tool for listing, creating, and organizing
the htpasswd files and their relationship with the 
auth_basic_input_file configuration of nginx. 
Currently builds everything using a config.toml file
specified for the website`,
		Run: func(cmd *cobra.Command, args []string) {
			config := readConfig("testConfig.toml")
			if err := organizeConfig(&config); err != nil {
				fmt.Println(err)
			}
			writeLocations(locationsPath, config.Groups)
			writeFiles(config.Groups)
		},
	}

	AuthCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "authConfig.toml", "the path to the TOML list of users and their groups")
	AuthCmd.PersistentFlags().StringVarP(&locationsPath, "locations", "l", "sites-enabled/default", "where nginx will look")
	AuthCmd.PersistentFlags().StringVarP(&passwordPath, "passwords", "p", "auth/.htpasswd", "defines location and prefix of passwods")

	AuthCmd.Execute()
}

func newGroup(g *Group, users []*User) {
	var cmd *exec.Cmd
	if len(users) > 0 {
		// create the new .htpasswd file
		cmd = exec.Command("htpasswd", "-bc", g.filename, users[0].name, users[0].Password)
		_, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}
		// now do the rest of the Users
		for _, User := range users[1:] {
			updateUser(g, User)
		}
	}
}

func updateUser(g *Group, User *User) {
	_, err := exec.Command("htpasswd", "-b", g.filename, User.name, User.Password).Output()
	if err != nil {
		log.Fatal(err)
	}
}

func deleteGroup(g *Group) {
	err := os.Remove(g.filename)
	if err != nil {
		log.Fatal(err)
	}
}

type config struct {
	Groups Groups
	Users  Users
}

func readConfig(filename string) config {
	var c config
	if _, err := toml.DecodeFile(filename, &c); err != nil {
		log.Fatal(err)
	}
	return c
}

func organizeConfig(c *config) error {
	// validation and organization of groups
	for gName, g := range c.Groups {
		g.name = gName
		g.filename = fmt.Sprintf("%s%s", passwordPath, g.name)
		for _, u := range c.Users {
			for _, ug := range u.Group {
				// add user to group if matches
				if ug == g.name || ug == Global {
					g.Users = append(g.Users, u)
				}
			}
		}

	}
	// validation and organization of users
outerloop:
	for uname, u := range c.Users {
		u.name = uname
		for _, ug := range u.Group {
			for _, g := range c.Groups {
				if ug == g.name || ug == Global {
					// check the next user
					continue outerloop
				}
			}
			// if no groups match user group
			return errors.New(fmt.Sprintf("%s group=%s doesn't match any existing groups", uname, u.Group))
		}
	}
	return nil
}

func location(location, filename string) string {
	str := `
	location %s {

        auth_basic "restricted";
        auth_basic_user_file /etc/nginx/auth/%s;
    }
`
	return fmt.Sprintf(str, location, filename)
}

func writeLocations(filename string, groups Groups) {
	lines, err := readLines(filename)
	if err != nil {
		log.Fatal(err)
	}

	var st int
	var en int
	// delete all entries within first
	for i, line := range lines {
		if strings.Contains(line, start) {
			st = i + 1
		}
		if strings.Contains(line, end) {
			en = i
		}
	}
	if st > 0 && en > 0 {
		// delete entries within
		lines = append(lines[:st], lines[en:]...)
	}
	// insert
	output := make([]string, 0)
	for _, g := range groups {
		output = append(output, location(g.Location, g.filename))
	}
	lines = append(lines[:st], append(output, lines[st:]...)...)
	// write
	err = writeLines(filename, lines)
	if err != nil {
		fmt.Println(err)
	}
}

func writeFiles(groups Groups) {
	for _, group := range groups {
		newGroup(group, group.Users)
	}
}

func readLines(file string) (lines []string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		const delim = '\n'
		line, err := r.ReadString(delim)
		if err == nil || len(line) > 0 {
			if err != nil {
				line += string(delim)
			}
			lines = append(lines, line)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return lines, nil
}

func writeLines(file string, lines []string) (err error) {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	for _, line := range lines {
		_, err := w.WriteString(line)
		if err != nil {
			return err
		}
	}
	return nil
}
