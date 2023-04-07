package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
)

// embedding the templates
var (
	//go:embed templates/header.html
	headerHTML string
	//go:embed templates/footer.html
	footerHTML string
)

type Author struct {
	Name, Bio string
	Links     []string
}

type Site struct {
	Name, Description, Link, License string
}

// paths
var (
	inFolder       = "./markdown"  // your markdown articles go in here
	outFolder      = "./output"    // your rendered html will end up here
	templateFolder = "./templates" // your header and footer go here
	pluginsFolder  = "./plugins"   // your plugins go here
	isWatching     = false         // whether we are watching any folders at launch
)

// config
var (

	// site config
	configFile = "site.config"

	// author vars
	author = Author{
		Name: "@donuts-are-good",
		Bio:  "open source enthusiast, author of bearclaw, professional coffee sipper and world-renowned pastry smuggler :)",
		Links: []string{
			"https://github.com/donuts-are-good/",
			"https://github.com/donuts-are-good/bearclaw",
		},
	}
	site = Site{
		Name:        "bearclaw blog",
		Description: "a blog about a tiny static site generator in Go!",
		// TODO: simplify this
		Link:    "https://" + "bearclaw.blog",
		License: "MIT License",
	}
)

func loadConfig() {

	// validate our config in memory
	// this will be bogus if you made changes that messed up your config
	if !validateConfig() {
		log.Fatal("could not validate in-memory config")
	}

	// check if config file exists
	_, err := os.Stat(configFile)

	if os.IsNotExist(err) {

		// if it doesn't exist, let's build it
		fmt.Println("No config file found, please enter the following information:")

		// prompt for username
		author.Name = promptUser("Author name (default: @donuts-are-good): ")
		if author.Name == "" {
			author.Name = "@donuts-are-good"
		}

		// prompt for author
		author.Bio = promptUser("Author bio (default: bearclaw author): ")
		if author.Bio == "" {
			author.Bio = "bearclaw author"
		}

		// prompt for author links
		author_links_string := promptUser("Author links (default: https://github.com/donuts-are-good/, https://github.com/donuts-are-good/bearclaw): ")
		if author_links_string == "" {
			author.Links = []string{
				"https://github.com/donuts-are-good/",
				"https://github.com/donuts-are-good/bearclaw",
			}
		} else {
			author.Links = strings.Split(author_links_string, ",")
		}

		// prompt for site name
		site.Name = promptUser("Site name (default: bearclaw blog): ")
		if site.Name == "" {
			site.Name = "bearclaw blog"
		}

		// prompt for site description
		site.Description = promptUser("Site description (default: a blog about a tiny static site generator in Go!): ")
		if site.Description == "" {
			site.Description = "a blog about a tiny static site generator in Go!"
		}

		// prompt for site link
		site.Link = promptUser("Site link (default: https://bearclaw.blog): ")
		if site.Link == "" {
			site.Link = "https://" + "bearclaw.blog"
		}

		// prompt for site license
		site.License = promptUser("Site license (default: MIT License): ")
		if site.License == "" {
			site.License = "MIT License"
		}

		// we're missing some config values here, but this is mainly
		// to test whether this way of doing it works.
		// since we've gathered some values, we'll now try to write.
		// if either of these fail, we should exit with non-zero
		// to satisfy the unix nerds :)

		// create the config file
		file, err := os.Create(configFile)
		if err != nil {
			log.Fatalf("could not create config file: %v", err)
		}
		defer file.Close()

		// write the config file
		config := []string{
			fmt.Sprintf("author_name: %s", author.Name),
			fmt.Sprintf("author_bio: %s", author.Bio),
			fmt.Sprintf("author_links: %s", author.Links),
			fmt.Sprintf("site_name: %s", site.Name),
			fmt.Sprintf("site_description: %s", site.Description),
			fmt.Sprintf("site_link: %s", site.Link),
			fmt.Sprintf("site_license: %s", site.License),
		}
		_, err = file.WriteString(strings.Join(config, "\n"))
		if err != nil {
			log.Fatalf("could not write to config file: %v", err)
		}

	} else {

		// validate our config on disk
		if !validateConfigFile(configFile) {
			log.Fatal("could not validate config on disk: ", configFile)
		}

		// read the config file
		// i think moritz said that os.Open was not the way to go
		// im sorry moritz
		file, err := os.Open(configFile)
		if err != nil {
			log.Fatalf("could not open config file: %v", err)
		}
		defer file.Close()

		// go through the lines of the config
		// each one should be a kv
		// i have no idea how we'll handle multiple links in the []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			kv := strings.SplitN(line, ":", 2)
			if len(kv) != 2 {
				continue
			}
			key, value := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			switch key {
			case "author_name":
				author.Name = value
			case "author_bio":
				author.Bio = value
			case "author_links":
				author.Links = strings.Split(value, ",")
				for i, link := range author.Links {
					author.Links[i] = strings.TrimSpace(link)
				}
			case "site_name":
				site.Name = value
			case "site_description":
				site.Description = value
			case "site_link":
				site.Link = value
				// site_links = strings.Split(value, ",")
				// for i, link := range site_links {
				// 	site_links[i] = strings.TrimSpace(link)
				// }
			case "site_license":
				site.License = value
			}
		}
	}
}

// promptUser will say a thing and prompt the user for a config value
func promptUser(message string) string {

	/*

		Bob Slydell: What would you say ya do here?

		Tom Smykowski: Well look, I already told you!
		I deal with the goddamn customers so the engineers don't have to!
		I have people skills! I am good at dealing with people! Can't you understand that?

		Tom Smykowski: What the hell is wrong with you people?

	*/

	// so we talk to the customer
	fmt.Print(message)

	// then we give it to the engineer
	input := bufio.NewScanner(os.Stdin)
	input.Scan()

	// what the hell is wrong with me
	return input.Text()
}

// validateConfig checks to see if any values have been
// loaded to the in-memory siteconfig values
func validateConfig() bool {

	// this could probably be better
	return !(author.Name == "" ||
		author.Bio == "" ||
		len(author.Links) == 0 ||
		site.Name == "" ||
		site.Description == "" ||
		site.Link == "" ||
		site.License == "")
}

// validateConfigFile is essentially the same as validateConfig, but
// it is checking the file itself and that it contains all of the fields
func validateConfigFile(siteConfigPath string) bool {

	// again, sorry moritz. ill update this later.
	// here we open the file
	file, err := os.Open(siteConfigPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// we make a new scanner
	scanner := bufio.NewScanner(file)

	// and we make a checklist of sorts
	configFound := map[string]bool{
		"author_name":      false,
		"author_bio":       false,
		"author_links":     false,
		"site_name":        false,
		"site_description": false,
		"site_link":        false,
		"site_license":     false,
	}

	// then we check line by line
	for scanner.Scan() {
		line := scanner.Text()

		// skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key, _ := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		if _, ok := configFound[key]; ok {
			configFound[key] = true
		}
	}

	// for all the ones we didn't find, mark an x
	for _, v := range configFound {
		if !v {
			return false
		}
	}

	return true
}

// scaffold will look for and/or create the necessary folders
func scaffold() {

	// we are making a list of folders here to check for the presence of
	// if they don't exist, we create them
	foldersToCreate := []string{inFolder, outFolder, templateFolder, pluginsFolder}
	createFoldersErr := createFolders(foldersToCreate)
	if createFoldersErr != nil {
		log.Fatalf("couldn't create a necessary folder: %v", createFoldersErr)
	}

}

// createFolders takes a list of folders and checks for them to exist, and creates them if they don't exist.
func createFolders(folders []string) error {
	for _, folder := range folders {
		if _, err := os.Stat(folder); os.IsNotExist(err) {

			err = os.MkdirAll(folder, os.ModePerm)
			if err != nil {
				return err
			}

			if folder == "templates" {

				err = recreateHeaderFooterFiles(folder)
				if err != nil {
					return err
				}
			}

		}
	}
	return nil
}
