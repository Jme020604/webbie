package main

// import all the needed packages
import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// create the global variables
var directories []string
var completeDirectories []string
var status200 []string
var full200 []string

// create the vars for the colors for in the stdout
const colorRed = "\033[0;31m"
const colorBlue = "\033[0;34m"
const colorGreen = "\033[0;32m"
const colorNone = "\033[0m"

func main() {
	// all the possible commands you can give the code
	Wordlist := flag.String("w", "", "Path to wordlist")
	url := flag.String("d", "", "Give the domain: https://example.com")
	show403 := flag.Bool("f", false, "show 403s")
	recursive := flag.Bool("r", false, "Work recursively on dirscan")
	workers := flag.Int("threads", 1000, "Give the ammount of threads")
	verbose := flag.Bool("v", false, "Print all processes (verbose)")
	output := flag.String("o", "", "Give output file to print to")

	flag.Parse() // gets all the values (like argv)

	fmt.Println("               _     _     _      ")
	fmt.Println("              | |   | |   (_)     ")
	fmt.Println(" __      _____| |__ | |__  _  ___ ")
	fmt.Println(" \\ \\ /\\ / / _ \\ '_ \\| '_ \\| |/ _ \\")
	fmt.Println("  \\ V  V /  __/ |_) | |_) | |  __/")
	fmt.Println("   \\_/\\_/ \\___|_.__/|_.__/|_|\\___| ")

	finalOutput := outputFile(*output) // runs outputFile to check for file

	if *Wordlist != "" { // checks if the wordlist is not empty
		fmt.Printf("Directory path: %s\n", *Wordlist)
		fmt.Printf("Number of threads%d\n", *workers)

		exstentionGuesser(*url, finalOutput)

		err := readDirs(*Wordlist, *url)
		if err != nil { // if the function returns an error it will quit the program
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Invalid choice. Please select a wordlist")
		os.Exit(1)
	}

	headerReader(*url, finalOutput)
	robots(*url, finalOutput)

	err := scanner(*show403, *recursive, *workers, *Wordlist, *verbose, finalOutput)
	if err != nil { // if the function returns an error it will quit the program
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// this functions make all the first directories by the wordlists
func readDirs(filePath string, url string) error {
	file, err := os.Open(filePath) // opens the file and if there occours an error it returns it
	if err != nil {
		return err
	}
	defer file.Close() // closes the file

	reader := io.Reader(file)
	buffer := make([]byte, 1024) // reads file and puts content in the buffer

	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		bufferStr := string(buffer[:n]) // reads the buffer
		lineArray := strings.Split(bufferStr, "\n")

		for _, line := range lineArray {
			if !strings.HasPrefix(line, "#") { // skips the lines that start with #
				directories = append(directories, line)
			}
		}
	}

	for _, directory := range directories { // goes thru the buffer and puts the full url in completeDirectories
		fullURL := fmt.Sprintf("%s/%s", url, directory)
		completeDirectories = append(completeDirectories, fullURL)
	}

	return nil
}

// this function reads some html header info
func headerReader(url string, output string) {
	response, err := http.Get(url) // sends get request to the site

	if err != nil { // if there was een error requesting the site it will return the error and exit the program
		fmt.Printf("Error sending Get request: %v\n", err)
		os.Exit(1)
	}

	// the html headers its searching for
	Server := response.Header.Get("Server")
	HSTS := response.Header.Get("Strict-Transport-Security")
	ContentType := response.Header.Get("Content-Type")

	// if ... is empty
	if Server == "" {
		Server = "Unkown"
	}
	if HSTS == "" {
		HSTS = "Not in use"
	}
	if ContentType == "" {
		ContentType = "Unkown"
	}
	if output != "" { // print to file
		if _, err := os.Stat(output); err == nil {
			// path/to/whatever exists
			f, err := os.OpenFile(output, os.O_APPEND|os.O_WRONLY, os.ModeAppend) // opens file
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			dt := time.Now()
			line1 := fmt.Sprintf("[%02d:%02d:%02d] Server: %s \n", dt.Hour(), dt.Minute(), dt.Second(), Server)
			line2 := fmt.Sprintf("[%02d:%02d:%02d] HTST: %s \n", dt.Hour(), dt.Minute(), dt.Second(), HSTS)
			line3 := fmt.Sprintf("[%02d:%02d:%02d] Content-Type: %s \n\n", dt.Hour(), dt.Minute(), dt.Second(), ContentType)

			//combines all the lines
			line := fmt.Sprintf("%s \n %s \n %s \n", line1, line2, line3)

			_, err = f.WriteString(line) // writes lines to the file
			if err != nil {
				log.Fatal(err)
			}
			//error handeling
		} else if errors.Is(err, os.ErrNotExist) {
			// path/to/whatever does *not* exist
			fmt.Printf("Error while writing to file, now exiting2")
			os.Exit(1)
		} else {
			fmt.Printf("Error while checking file, now exiting3")
			os.Exit(1)
		}
	} else {
		//prints all the info to the terminal
		dt := time.Now()
		fmt.Printf("[%s%02d:%02d:%02d%s] Server: %s \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, Server)
		fmt.Printf("[%s%02d:%02d:%02d%s] HTST: %s \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, HSTS)
		fmt.Printf("[%s%02d:%02d:%02d%s] Content-Type: %s \n\n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, ContentType)
	}
}

// function that activates the workers
func scanner(show403 bool, recursive bool, workers int, path string, verbose bool, output string) error {
	var wg sync.WaitGroup                // creates waitgoup so all workers are able to comminucate
	urlQueue := make(chan string, 10000) // makes a queue for all the urls

	for i := 0; i < workers; i++ { // creates an x amount of workers and adds them to the waitgoup
		wg.Add(1)
		go worker(&wg, urlQueue, show403, recursive, verbose, output)
	}

	for _, URL := range completeDirectories {
		urlQueue <- URL // puts the url in the queue
		if recursive {
			// Enqueue subdirectories for scanning
			subDirs, err := getSubdirectories(URL)
			if err != nil {
				fmt.Printf("Error getting subdirectories for %s: %v\n", URL, err)
				continue
			}
			for _, subDir := range subDirs {
				urlQueue <- subDir // adding the subdirs in the url queue
			}
		}
	}

	close(urlQueue) // if all is scanned the queue closes
	wg.Wait()       // waits for all workers to finish

	return nil
}

// function that gets all the subdirectories from the site
func getSubdirectories(url string) ([]string, error) {
	response, err := http.Get(url) // sends get request to the site
	if err != nil {
		return nil, err
	}
	defer response.Body.Close() // closes the connetion

	var subDirs []string
	tokenizer := html.NewTokenizer(response.Body) // turns html into 'tokens'

	for {
		tokenType := tokenizer.Next() // for loop that gous thru all the toekns

		switch tokenType {
		case html.ErrorToken:
			return subDirs, nil
		case html.StartTagToken, html.SelfClosingTagToken: // if the token looks like <..> or <../>
			token := tokenizer.Token()

			if response.StatusCode == 200 { // if the get request sends an status code 200 (okay), it makes full urls of the subdirs
				for _, start := range directories {
					line := fmt.Sprintf("%s/%s", url, start)
					subDirs = append(subDirs, line)
				}
			}
			if token.Data == "a" { //if the token contains a (link in html)
				for _, attr := range token.Attr {
					if attr.Key == "href" && strings.HasPrefix(attr.Val, "/") { // checks for the link in <a href="example.com/dir">....</a> gives example.com/dir
						line := fmt.Sprintf("%s%s", url, attr.Val)
						subDirs = append(subDirs, line)
					}
				}
			}
		}
	}
}

// this function defines a worker
func worker(wg *sync.WaitGroup, urlQueue <-chan string, show403, recursive bool, verbose bool, output string) {
	defer wg.Done() // sends 'done' in the waitgroup is the worker is done with its job

	for URL := range urlQueue {
		response, err := http.Get(URL) // sends get request

		if err != nil {
			fmt.Printf("Error sending Get request: %v\n", err)
			continue
		}
		defer response.Body.Close() // closes the connection

		if output != "" { // checks if 'output' is not empty
			if _, err := os.Stat(output); err == nil {
				// path/to/whatever exists
				f, err := os.OpenFile(output, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
				if err != nil {
					log.Fatal(err)
				}
				defer f.Close()

				dt := time.Now()
				line := fmt.Sprintf("[%02d:%02d:%02d][%s] %s  \n", dt.Hour(), dt.Minute(), dt.Second(), response.Status, URL)

				_, err = f.WriteString(line) // writes line to file
				if err != nil {
					log.Fatal(err)
				}
			} else if errors.Is(err, os.ErrNotExist) { // error handeling
				// path/to/whatever does *not* exist
				fmt.Printf("Error while writing to file, now exiting2")
				os.Exit(1)
			} else {
				fmt.Printf("Error while checking file, now exiting3")
				os.Exit(1)
			}
		} else {
			// these if else statements are for handling the output corectly
			if !verbose {
				if response.StatusCode != 404 {
					if !show403 {
						if response.StatusCode != 403 {
							dt := time.Now()
							status200 = append(status200, URL)
							fmt.Printf("[%s%02d:%02d:%02d%s][%s%s%s] %s  \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, colorGreen, response.Status, colorNone, URL)
						}
					} else {
						if !recursive {
							if response.StatusCode == 403 {
								dt := time.Now()
								fmt.Printf("[%s%02d:%02d:%02d%s][%s%s%s] %s  \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, colorRed, response.Status, colorNone, URL)
							} else {
								status200 = append(status200, URL)
								dt := time.Now()
								fmt.Printf("[%s%02d:%02d:%02d%s][%s%s%s] %s  \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, colorGreen, response.Status, colorNone, URL)
							}
						} else {
							if response.StatusCode == 403 {
								dt := time.Now()
								fmt.Printf("[%s%02d:%02d:%02d%s][%s%s%s] %s  \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, colorRed, response.Status, colorNone, URL)
							} else if response.StatusCode == 200 {
								status200 = append(status200, URL)
								dt := time.Now()
								fmt.Printf("[%s%02d:%02d:%02d%s][%s%s%s] %s  \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, colorGreen, response.Status, colorNone, URL)
							}
						}
					}
				}
			} else {
				if response.StatusCode == 403 {
					dt := time.Now()
					fmt.Printf("[%s%02d:%02d:%02d%s][%s%s%s] %s  \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, colorRed, response.Status, colorNone, URL)
				} else if response.StatusCode == 404 {
					dt := time.Now()
					fmt.Printf("[%s%02d:%02d:%02d%s][%s%s%s] %s  \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, colorRed, response.Status, colorNone, URL)
				} else {
					status200 = append(status200, URL)
					dt := time.Now()
					fmt.Printf("[%s%02d:%02d:%02d%s][%s%s%s] %s  \n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, colorGreen, response.Status, colorNone, URL)
				}
			}
		}

	}
}

// this function tries to determen the exstention of the index page
func exstentionGuesser(url string, output string) error {
	extentions := [4]string{"index.html", "index.php", "index.htm", "index.xhtml"} // possible index pages

	for _, ext := range extentions {
		goodURL := fmt.Sprintf("%s/%s", url, ext) // makes a propper url
		response, err := http.Get(goodURL)        // send get request
		if err != nil {
			fmt.Printf("Error sending Get request: %v\n", err)
			return err
		}
		defer response.Body.Close() // closes the connection

		if response.StatusCode == 200 { // when the response is okay
			if output != "" { // writes to file
				if _, err := os.Stat(output); err == nil {
					// path/to/whatever exists
					f, err := os.OpenFile(output, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
					if err != nil {
						log.Fatal(err)
					}
					defer f.Close()

					dt := time.Now()
					line := fmt.Sprintf("[%02d:%02d:%02d] This site uses %s as its index page\n", dt.Hour(), dt.Minute(), dt.Second(), ext)

					_, err = f.WriteString(line)
					if err != nil {
						log.Fatal(err)
					}
				} else if errors.Is(err, os.ErrNotExist) { // error handeling
					// path/to/whatever does *not* exist
					fmt.Printf("Error while writing to file, now exiting2")
					os.Exit(1)
				} else {
					fmt.Printf("Error while checking file, now exiting3")
					os.Exit(1)
				}
			} else { // else print to terminal
				dt := time.Now()
				fmt.Printf("[%s%02d:%02d:%02d%s] This site uses %s as its index page\n", colorBlue, dt.Hour(), dt.Minute(), dt.Second(), colorNone, ext)
				break
			}
		}
	}

	return nil
}

// this function read the robots.txt of a site
func robots(url string, output string) {
	// Construct the URL for robots.txt
	URL := fmt.Sprintf("%s/robots.txt", url) // creates the proper url

	response, err := http.Get(URL) // sends get request
	if err != nil {
		fmt.Printf("Error sending GET request: %v\n", err)
		os.Exit(1)
	}
	defer response.Body.Close() // Close the response body when done

	if response.StatusCode != http.StatusOK { // checks the response
		fmt.Printf("Received a non-200 status code for robots.txt: %s\n", response.Status)
	}

	if output != "" { // writes to file
		if _, err := os.Stat(output); err == nil {
			// path/to/whatever exists
			f, err := os.OpenFile(output, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			fmt.Println("The contents of robots.txt")

			data, err := io.Copy(f, response.Body)
			if err != nil {
				fmt.Printf("Error reading robots.txt: %v\n", err)
			}

			line := fmt.Sprintf("Data Length: %d bytes\n\n", data)

			_, err = f.WriteString(line)
			if err != nil {
				log.Fatal(err)
			}
		} else if errors.Is(err, os.ErrNotExist) { // error handeling
			// path/to/whatever does *not* exist
			fmt.Printf("Error while writing to file, now exiting2")
			os.Exit(1)
		} else {
			fmt.Printf("Error while checking file, now exiting3")
			os.Exit(1)
		}
	} else { // writes to terminal
		fmt.Println("The contents of robots.txt")

		data, err := io.Copy(os.Stdout, response.Body) // Copy the response body to os.Stdout to print the contents
		if err != nil {
			fmt.Printf("Error reading robots.txt: %v\n", err)
		}
		fmt.Printf("Data Length: %d bytes\n\n", data)
	}
}

// function that checks file or makes one
func outputFile(output string) string {
	if output != "" {
		if _, err := os.Stat(output); err == nil {
			// path/to/whatever exists
			fmt.Printf("File exists, writing to file... \n")
		} else if errors.Is(err, os.ErrNotExist) {
			// path/to/whatever does *not* exist
			f, err := os.Create("./outputWebbie.txt") // creates file
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			output = "./outputWebbie.txt" // change value to write to the file
			fmt.Printf("%sGiven file did not exist%s,  './outputWebbie.txt' created, writing to file... \n", colorRed, colorNone)
		} else {
			fmt.Printf("Error while checking file, now exiting 1")
			os.Exit(1)
		}
		return output
	} else {
		return ""
	}

}
