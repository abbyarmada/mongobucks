package slack

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/c0nrad/mongobucks/models"
)

type Handler struct {
	Re          *regexp.Regexp
	HandlerFunc func(command string, vars map[string]string) string
}

var Handlers []Handler

func init() {
	Handlers = BuildHandlers()
}

// mongobucks: balance
//   # Respond to use with current balance
// mongobucks: give <username> 10
//   # Transfer 10 mongobucks from sender to <username>
// mongobucks: rain 10
//   # Evenly split 10 mongobucks amongst chat room
// mongobucks: help
//   # Display help room

func BuildHandlers() []Handler {
	handler := []Handler{}
	handler = append(handler, Handler{regexp.MustCompile("^(balance|b)$"), BalanceHandler})
	handler = append(handler, Handler{regexp.MustCompile("^(give|g) (?P<to>.*) (?P<amount>[0-9]*) (?P<memo>.*)$"), TransferHandler})
	handler = append(handler, Handler{regexp.MustCompile("^(give|g) (?P<to>.*) (?P<amount>[0-9]*)$"), TransferHandler})
	handler = append(handler, Handler{regexp.MustCompile("^(balance|b) all$"), AllBalanceHandler})

	return handler
}

func HandleMessage(message Message) string {

	text := strings.Join(strings.Fields(message.Text)[1:], " ")

	for _, Handler := range Handlers {
		if Handler.Re.MatchString(text) {
			names := Handler.Re.SubexpNames()
			values := Handler.Re.FindAllStringSubmatch(text, -1)[0]
			values = values[1:]

			vars := map[string]string{}
			for i, value := range values {
				name := names[i+1]
				if name != "" {
					vars[names[i+1]] = value
				}
			}

			username, err := GetUsername(message.User)
			if err != nil {
				return err.Error()
			}
			vars["user"] = username
			vars["channel"] = message.Channel
			return Handler.HandlerFunc(text, vars)

		}
	}

	return "[-] Command not recognized. Use 'help' for available commands."
}

func TrimUsername(username string) string {
	username = strings.Replace(username, "<@", "", -1)
	username = strings.Replace(username, ">", "", -1)
	return username
}

func BalanceHandler(command string, vars map[string]string) string {
	fmt.Println("BalanceHandler", vars)

	balance, err := models.GetBalance(vars["user"])
	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf("%d mongobucks", balance)
}

func AllBalanceHandler(command string, vars map[string]string) string {

	users, err := models.GetUsers()
	if err != nil {
		return err.Error()
	}

	out := "Balances: \n"
	for _, u := range users {
		out += "@" + u.Username + ": " + strconv.Itoa(u.Balance) + "\n"
	}

	return out
}

func TransferHandler(command string, vars map[string]string) string {
	fmt.Println("[+] TransferHandler", vars)

	if !strings.HasPrefix(vars["to"], "<@") {
		return "[-] Prefix the username with '@', for example '@stuart'"
	}

	from := vars["user"]
	to, err := GetUsername(TrimUsername(vars["to"]))
	if err != nil {
		return "invalid user: " + err.Error()
	}

	amount, err := strconv.Atoi(vars["amount"])
	if err != nil {
		return "invalid amount: " + err.Error()
	}

	out, err := models.ExecuteTransfer(from, to, amount, vars["memo"])
	if err != nil {
		return err.Error()
	}

	return out
}