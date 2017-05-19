package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/chzyer/readline"
	"github.com/sartura/go-netconf/netconf"
)

func usage(w io.Writer) {
	io.WriteString(w, "commands:\n")
	io.WriteString(w, completer.Tree("    "))
}

var completer = readline.NewPrefixCompleter(
	readline.PcItem("mode",
		readline.PcItem("vi"),
		readline.PcItem("emacs"),
	),
	readline.PcItem("login"),
	readline.PcItem("logout"),
	readline.PcItem("get"),
	readline.PcItem("get-config"),
	readline.PcItem("set"),
	readline.PcItem("datastore"),
	readline.PcItem("quit"),
)

func main() {
	//libyang

	// set error callback for libyang
	ctx, s := getNetconfContext()

	l, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[31m»\033[0m ",
		HistoryFile:     "/tmp/readline.tmp",
		AutoComplete:    completer,
		InterruptPrompt: "\nInterrupt, Press Ctrl+D to exit",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	var datastore string
	var username string
	var password []byte
	var ip string
	var port string

	datastore = "running"

	setPasswordCfg := l.GenPasswordConfig()
	setPasswordCfg.SetListener(func(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
		l.SetPrompt(fmt.Sprintf("Enter password(%v): ", len(line)))
		l.Refresh()
		return nil, 0, false
	})

	log.SetOutput(l.Stderr())
	for {
		line, err := l.Readline()
		if err == io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "mode "):
			switch line[5:] {
			case "vi":
				l.SetVimMode(true)
			case "emacs":
				l.SetVimMode(false)
			default:
				println("invalid mode:", line[5:])
			}
		case strings.HasPrefix(line, "get "):
			if s == nil {
				print("Please login first\n")
				break
			}
			xpath := strings.TrimSpace(line[4:])

			err := netconfOperation(s, ctx, datastore, xpath, nil, "get")
			if err != nil {
				println("ERROR: ", err)
			}
		case strings.HasPrefix(line, "get-config "):
			if s == nil {
				print("Please login first\n")
				break
			}
			xpath := strings.TrimSpace(line[11:])

			err := netconfOperation(s, ctx, datastore, xpath, nil, "get-config")
			if err != nil {
				println("ERROR: ", err)
			}
		case strings.HasPrefix(line, "set "):
			if s == nil {
				print("Please login first\n")
				break
			}
			xpath := strings.TrimSpace(line[4:])

			err := netconfOperation(s, ctx, datastore, xpath, nil, "set")
			if err != nil {
				println("ERROR: ", err)
			}
		case line == "mode":
			if l.IsVimMode() {
				println("current mode: vim")
			} else {
				println("current mode: emacs")
			}
		case strings.HasPrefix(line, "datastore "):
			datastoreInput := strings.TrimSpace(line[10:])

			switch {
			case datastoreInput == "startup":
				datastore = datastoreInput
			case datastoreInput == "running":
				datastore = datastoreInput
			case datastoreInput == "candidate":
				datastore = datastoreInput
			default:
				print("invalid datastore!\n")
			}

		case line == "logout":
			cleanNetconfContext(ctx, s)
			ctx = nil
			s = nil
		case line == "login":
			if s != nil {
				print("you are already loged in\n")
				break
			}
			var auth *ssh.ClientConfig

			// hard code loginn
			/*
				l.SetPrompt("username: ")
				username, err = l.Readline()
				if err != nil {
					goto login_fail
				}
				password, err = l.ReadPassword("password: ")
				if err != nil {
					goto login_fail
				}
				l.SetPrompt("ip: ")
				ip, err = l.Readline()
				if err != nil {
					goto login_fail
				}
				l.SetPrompt("port: ")
				port, err = l.Readline()
				if err != nil {
					goto login_fail
				}
			*/
			username = "netconf"
			password = []byte("netconf")
			ip = "localhost"
			port = "830"

			// create new libyang context with the remote yang files
			l.SetPrompt("\033[31m»\033[0m ")
			auth = netconf.SSHConfigPassword(username, string(password))
			s, err = netconf.DialSSH(ip+":"+port, auth)
			if err != nil {
				goto login_fail
			}

			// create new libyang context with the remote yang files
			ctx, err = getRemoteContext(s)
			if err != nil {
				s.Close()
				s = nil
				goto login_fail
			}

			break
		login_fail:
			l.SetPrompt("\033[31m»\033[0m ")
			print("login failed\n")
			break
		case line == "help":
			usage(l.Stderr())
		case line == "quit":
			goto exit
		case line == "":
		default:
			print("Invalid command.\n")
		}
	}
exit:

	cleanNetconfContext(ctx, s)
}
