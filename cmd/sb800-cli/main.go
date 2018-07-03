package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/urfave/cli"
)

func switchBoxURL(url string) (string, error) {
	home := os.Getenv("HOME")
	configDir := home + "/.config/sb800-cli/"
	configFile := configDir + "/config"

	if url != "" {

		//Save URL to disk
		os.Mkdir(configDir, 0700)
		err := ioutil.WriteFile(configFile, []byte(url), 0644)
		if err != nil {
			return "", fmt.Errorf("switchBoxURL: %v", err)
		}
		return url, nil
	}

	// Try to read the url from config file
	b, err := ioutil.ReadFile(configFile) // just pass the file name
	if err != nil {
		return "", fmt.Errorf("switchBoxURL: %v", err)
	}
	s := string(b)
	return s, nil
}

func printByteReverse(s string) {
	d, _ := strconv.ParseInt(s, 16, 64)

	var out []rune
	for _, b := range fmt.Sprintf("%08b", d) {
		out = append([]rune{' ', b}, out...)
	}
	fmt.Println(string(out))

}

func main() {

	app := cli.NewApp()
	app.Name = "sb800-cli"
	app.Usage = `cli for SwitchBox800`
	app.HideHelp = true
	app.Version = "0.3"
	app.EnableBashCompletion = true

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "url",
			Usage: "Url of switchbox800 cli",
		},
		cli.BoolFlag{
			Name:  "status",
			Usage: "Print status then quit",
		},
		cli.IntFlag{
			Name:  "position",
			Value: -1,
			Usage: "Position of switch to perform action on",
		},
		cli.IntFlag{
			Name:  "sleep",
			Value: 500,
			Usage: "Time sleep to sleep during reset or off/on action",
		},
		cli.BoolFlag{
			Name:  "on",
			Usage: "Turn switch on",
		},
		cli.BoolFlag{
			Name:  "off",
			Usage: "Turn switch off",
		},
		cli.BoolFlag{
			Name:  "reset",
			Usage: "Turn switch off, sleep 500 ms (default), then turn switch on",
		},
	}
	app.Action = func(c *cli.Context) {

		url, err := switchBoxURL(c.String("url"))

		if err != nil {
			log.Fatal(err)
		}

		// Setup client
		sb := SwitchBox{
			client: &http.Client{Timeout: time.Second * 10},
			url:    url,
		}

		if c.Bool("status") {
			sb.showStatus()
			os.Exit(0)
		}

		if c.Int("position") < 0 || c.Int("position") > 8 {
			fmt.Println("Position must be 1-8")
			os.Exit(1)
		}

		var position uint = uint(c.Int("position"))
		if !c.Bool("on") && !c.Bool("reset") && !c.Bool("off") {
			fmt.Println("Please specify --on, --off or --reset")
			os.Exit(1)
		}

		fmt.Println("\nConnecting to SwitchBox800 (" + url + ")")

		if c.Bool("off") || c.Bool("reset") {
			sb.turnOff(position)
		}

		if c.Bool("off") && c.Bool("on") || c.Bool("reset") {
			fmt.Printf("\n\t(Sleep %d ms)\n", c.Int("sleep"))
			time.Sleep(time.Duration(c.Int("sleep")) * time.Millisecond)
			time.Sleep(time.Duration(c.Int("sleep")) * time.Millisecond)
		}

		if c.Bool("on") || c.Bool("reset") {
			sb.turnOn(position)
		}
	}
	app.Run(os.Args)

}

type SwitchBox struct {
	client *http.Client
	url    string
}

func (sb SwitchBox) updateSwitchBox(s string) {

	printHeader()
	fmt.Print("Status before:\t")
	sb.showStatusShort()

	resp, err := sb.client.Get("http://" + sb.url + "/k1" + s)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("Status after:\t")
	printByteReverse(string(body[0:2]))
	fmt.Println("")
}

func (sb SwitchBox) turnOn(p uint) {
	turnOnString := fmt.Sprintf("%02x000000", 1<<(p-1))
	sb.updateSwitchBox(turnOnString)
}

func (sb SwitchBox) turnOff(p uint) {
	turnOffString := fmt.Sprintf("00%02x0000", 1<<(p-1))
	sb.updateSwitchBox(turnOffString)
}

func (sb SwitchBox) showStatusShort() {
	resp, err := sb.client.Get("http://" + sb.url + "/k0")

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	printByteReverse(string(body[0:2]))
}

func (sb SwitchBox) showStatus() {
	resp, err := sb.client.Get("http://" + sb.url + "/k0")

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	printHeader()
	fmt.Print("Current status:\t")
	printByteReverse(string(body[0:2]))

	fmt.Print("Reset ongoing:\t")
	printByteReverse(string(body[2:4]))

	fmt.Print("Read possible:\t")
	printByteReverse(string(body[4:6]))

	fmt.Print("Write possible:\t")
	printByteReverse(string(body[6:8]))

	fmt.Print("\nBox reserved by other user:\t")
	fmt.Println(body[8] == 1)
}

func printHeader() {
	fmt.Print("\n               \t 1 2 3 4 5 6 7 8")
	fmt.Print("\n               \t ---------------\n")
}
