package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/russross/blackfriday"
)

// embedding the templates
var (
	//go:embed templates/header.html
	headerHTML string
	//go:embed templates/footer.html
	footerHTML string
)

// paths
var (
	inFolder       = "./markdown"  // your markdown articles go in here
	outFolder      = "./output"    // your rendered html will end up here
	templateFolder = "./templates" // your header and footer go here
	pluginsFolder  = "./plugins"   // your plugins go here

)

// init runs before main()
func init() {

	// we are making a list of folders here to check for the presence of
	// if they don't exist, we create them
	foldersToCreate := []string{inFolder, outFolder, templateFolder, pluginsFolder}
	createFoldersErr := createFolders(foldersToCreate)
	if createFoldersErr != nil {
		log.Println(createFoldersErr)
	}

}

func main() {

	// check to see if the user ran with --watch
	watchFlag := flag.Bool("watch", false, "watch the current directory for changes")
	flag.Parse()

	// if they did...
	if *watchFlag {

		// make a list of folders to keep an eye on
		foldersToWatch := []string{templateFolder, inFolder}

		// the process to watch it goes in a goroutine
		go watchFoldersForChanges(foldersToWatch)

		// give the user some type of confirmation
		fmt.Println("Waiting for changes: ", foldersToWatch)

		// select {} is a blocking operation that keeps
		// the program from closing
		select {}
	}

	// now, if nothing has gone wrong, we process the html
	markdownToHTML(inFolder, outFolder, templateFolder)
	createPostList(inFolder, outFolder, templateFolder)

}

// recreateHeaderFooterFiles recreates the header and footer files
// if we're rebuilding the templates, we'll need these.
func recreateHeaderFooterFiles(templatesFolder string) error {
	headerFile, err := os.Create(templatesFolder + "/header.html")
	if err != nil {
		return err
	}
	defer headerFile.Close()
	_, err = headerFile.WriteString(headerHTML)
	if err != nil {
		return err
	}
	footerFile, err := os.Create(templatesFolder + "/footer.html")
	if err != nil {
		return err
	}
	defer footerFile.Close()
	_, err = footerFile.WriteString(footerHTML)
	if err != nil {
		return err
	}

	return nil
}

func watchFoldersForChanges(folders []string) {

	// range through the watched files
	for _, folder := range folders {

		// create a new watcher for each watched folder
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}

		// don't forget to close it
		defer watcher.Close()

		// make a channel for goroutine messaging
		done := make(chan bool)
		go func() {

			// for { ev ... ver }
			for {
				select {
				case event := <-watcher.Events:
					if event.Op&fsnotify.Write == fsnotify.Write {

						// if there's an event, remark and rebuild
						log.Println("modified:", event.Name, " - rebuilding files..")
						markdownToHTML(inFolder, outFolder, templateFolder)
						createPostList(inFolder, outFolder, templateFolder)

					}

					// if we get an error instead of event, announce it
				case err := <-watcher.Errors:
					log.Println("error:", err)

				}
			}
		}()

		err = watcher.Add(folder)
		if err != nil {
			log.Fatal(err)
		}
		<-done
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

// markdownToHTML converts markdown documents to HTML
func markdownToHTML(inFolder, outFolder, templateFolder string) {
	files, _ := os.ReadDir(inFolder)

	for _, file := range files {

		// only markdown files
		if filepath.Ext(file.Name()) == ".md" {

			// open the selected markdown file
			markdownFile, _ := os.Open(inFolder + "/" + file.Name())

			// don't forget to close it when done
			defer markdownFile.Close()

			// create the html file
			htmlFile, _ := os.Create(outFolder + "/" + file.Name() + ".html")
			defer htmlFile.Close()

			// read the md
			reader := bufio.NewReader(markdownFile)
			markdown, _ := io.ReadAll(reader)

			// send the md to blackfriday
			html := blackfriday.MarkdownCommon(markdown)

			// read in our templates
			header, _ := os.ReadFile(templateFolder + "/header.html")
			footer, _ := os.ReadFile(templateFolder + "/footer.html")

			// assemble in order
			completeHTML := string(header) + strings.TrimSpace(string(html)) + string(footer)

			// pass the assembled html into ScanForPluginCalls
			htmlAfterPlugins, htmlAfterPluginsErr := ScanForPluginCalls(completeHTML)
			if htmlAfterPluginsErr != nil {
				log.Println("error inserting plugin content: ", htmlAfterPluginsErr)
				log.Println("returning content without plugin...")

				// if there's an error, let's just take the html from before the error and use that.
				htmlAfterPlugins = completeHTML
			}

			fmt.Fprintln(htmlFile, htmlAfterPlugins)
		}
	}
}

// createPostList creates the page that has a list of all of your posts
func createPostList(inFolder, outFolder, templateFolder string) {

	// read the files in the directory
	files, _ := os.ReadDir(inFolder)

	// sort them by mod time
	sort.Slice(files, func(i, j int) bool {
		fi, _ := os.Stat(inFolder + "/" + files[i].Name())
		fj, _ := os.Stat(inFolder + "/" + files[j].Name())
		return fi.ModTime().After(fj.ModTime())
	})

	// unordered list
	postList := "<ul>"

	// for all files...
	for _, file := range files {

		// if it is a markdown file...
		if filepath.Ext(file.Name()) == ".md" {

			// get the filename/title
			title := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

			// put it on the list with the html
			// postList += "<li><a href='" + file.Name() + ".html'>" + title + "</a></li>"
			postList += "<li><a href='" + url.PathEscape(file.Name()) + ".html'>" + title + "</a></li>"

		}

	}

	// if there are more than 0 posts make an RSS feed
	if len(files) > 0 {

		// generate the rss feed
		CreateXMLRSSFeed(inFolder, outFolder)

	}

	// end the list
	postList += "</ul>"

	// create the posts file
	htmlFile, _ := os.Create(outFolder + "/posts.html")

	// don't forget to close it
	defer htmlFile.Close()

	// read the header/footer templates
	header, _ := os.ReadFile(templateFolder + "/header.html")
	footer, _ := os.ReadFile(templateFolder + "/footer.html")

	// combine them
	fmt.Fprintln(htmlFile, string(header)+postList+string(footer))
}
