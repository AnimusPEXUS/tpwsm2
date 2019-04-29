package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/AnimusPEXUS/utils/environ"

	"golang.org/x/crypto/ssh/terminal"
)

const (
	HELP_TEXT = `
  !h, !help     - help

  !l            - list
  !d name       - delete
  !n name name2 - rename

  !s            - save
  !r            - change password
  !quit, !exit  - exit (Ctrl+d also)

  other_text    - used as name - start editing existing record.
                  if prefixed with '+' - create if not exists.
`
	STORAGE_FN = "data.db"
	TMP_FN     = "tpwsm.tmp.fn"
)

func useLess(txt string) error {

	b := bytes.NewBuffer([]byte(txt))

	c := exec.Command("less")
	c.Stdin = b
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return err
	}

	return nil
}

func displayHidden(txt string, filename string) (string, error) {

	e := environ.NewFromStrings(os.Environ())
	editor := e.Get("EDITOR", "mcedit")

	fn := path.Base(filename)

	if fn == STORAGE_FN {
		return "", errors.New("unacceptable name")
	}

	err := ioutil.WriteFile(fn, []byte(txt), 0700)
	if err != nil {
		return "", err
	}

	c := exec.Command(editor, fn)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	no_need_to_delete := make(chan bool, 1)

	go func(no_need_to_delete chan bool, fn string) {
		log.Print("scheduled automatic " + fn + " delete")

		select {
		case <-no_need_to_delete:
			log.Print("automatic " + fn + " delete canceled")
			return
		case <-time.After(time.Second * 10):
			log.Print("timedout. deleting " + fn)
			err = os.Remove(fn)
			if err != nil {
				log.Print(err)
			}
		}

	}(no_need_to_delete, fn)

	err = c.Run()
	if err != nil {
		return "", err
	}

	no_need_to_delete <- true

	d, err := ioutil.ReadFile(fn)
	if err != nil {
		return "", err
	}

	err = os.Remove(fn)
	if err != nil {
		return "", err
	}

	return string(d), nil
}

func askPass(prompt string) (string, error) {
	fmt.Printf("%s", prompt)
	defer fmt.Printf("\n")

	res, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}

	return string(res), nil
}

func WriteEncFile(filename string, passwds map[string]string, passwd string) error {

	fmt.Println("writting " + filename)

	os.Rename(filename, filename+".backup")

	plaintext, err := json.MarshalIndent(passwds, "  ", "  ")
	if err != nil {
		return err
	}

	h := sha256.New()
	h.Write([]byte(passwd))

	block, err := aes.NewCipher(h.Sum([]byte{}))
	if err != nil {
		return err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, bytes.NewReader(ciphertext))
	if err != nil {
		return err
	}

	return nil

}

func Save(passwds map[string]string, passwd string) error {
	err := WriteEncFile(STORAGE_FN, passwds, passwd)
	if err != nil {
		return err
	}
	return nil
}

func ReadEncFile(filename string, passwd string) (map[string]string, error) {

	ciphertext, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(passwd))

	block, err := aes.NewCipher(h.Sum([]byte{}))
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	stream.XORKeyStream(ciphertext, ciphertext)

	ret := make(map[string]string)

	err = json.Unmarshal(ciphertext, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func main() {

	fmt.Printf("")

	password := ""
	if p, err := askPass("Password?: "); err != nil {
		panic(err)
	} else {
		password = p
	}

	// password_e := url.QueryEscape(password)

	// values := &url.Values{
	// 	"_key": []string{password},
	// }

	// fmt.Println("values:", values.Encode())

	// values_e := values.Encode()

	// fmt.Println("password:", password)
	// fmt.Println("values_e:", values_e)

	fmt.Println("Trying to open file '" + STORAGE_FN + "'")
	passwords, err := ReadEncFile(STORAGE_FN, password)
	if err != nil {
		panic(err)
	}

	defer func() {
		Save(passwords, password)
		fmt.Println("Bye!")
	}()

	fmt.Println(HELP_TEXT)

	reader := bufio.NewReader(os.Stdin)

loo:
	for {

		fmt.Print("> ")
		command, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Printf("\n")
				break
			}
			fmt.Println("error: " + err.Error())
			continue
		}

		command = strings.TrimRight(command, "\n")

		command_splitted := strings.Split(command, " ")

		if len(command_splitted[0]) > 0 && command_splitted[0][0] != '!' {

			name := command_splitted[0]

			plus := strings.HasPrefix(name, "+")
			if plus {
				name = name[1:]
			}

			if path.Base(name) == STORAGE_FN {
				fmt.Println("error: unacceptable name")
				continue
			}

			_, ok := passwords[name]
			if !ok {
				if plus {
					passwords[name] = ""
				} else {
					fmt.Println("error: no such record")
					continue
				}

			}

			d, err := displayHidden(passwords[name], name+"."+TMP_FN)
			if err != nil {
				fmt.Println("error: " + err.Error())
				continue
			}

			passwords[name] = string(d)

			continue
		}

		switch command_splitted[0] {
		default:
			fmt.Println("error: command not supported")
			continue
		case "!h":
			fallthrough
		case "!help":
			fmt.Println(HELP_TEXT)

		case "!s":
			err = Save(passwords, password)
			if err != nil {
				fmt.Println("error: " + err.Error())
				continue
			}
			continue

		case "!l":
			if len(command_splitted) != 1 {
				fmt.Println("no params")
				continue
			}

			l := make([]string, 0)

			for k, _ := range passwords {
				l = append(l, k)
			}

			sort.Strings(l)

			l2 := ""
			for _, l := range l {
				l2 += fmt.Sprintf(" %s\n", l)
			}

			err = useLess(l2)
			if err != nil {
				fmt.Println("error: " + err.Error())
				continue
			}

		case "!d":
			if len(command_splitted) != 2 {
				fmt.Println("id required")
				continue
			}

			delete(passwords, command_splitted[1])

		case "!n":
			if len(command_splitted) != 3 {
				fmt.Println("id and name required")
				continue
			}

			passwords[command_splitted[2]] = passwords[command_splitted[1]]

			delete(passwords, command_splitted[1])

		case "!r":
			if len(command_splitted) != 1 {
				fmt.Println("no params")
				continue
			}

			password1 := ""
			password2 := ""
			if p, err := askPass("RePassword?: "); err != nil {
				fmt.Println("error: " + err.Error())
				continue
			} else {
				password1 = p
			}

			if p, err := askPass("confirm: "); err != nil {
				fmt.Println("error: " + err.Error())
				continue
			} else {
				password2 = p
			}

			if password1 != password2 {
				fmt.Println("error: missmatch")
				continue
			}

			password = password1

		case "!exit":
			fallthrough
		case "!quit":
			break loo
		}
	}

}
