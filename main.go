package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	_ "github.com/xeodou/go-sqlcipher"

	"github.com/AnimusPEXUS/utils/environ"

	"github.com/jinzhu/gorm"

	"golang.org/x/crypto/ssh/terminal"
)

const (
	HELP_TEXT = `
  !h, !help    - help

  !l           - list
  !d id        - delete
  !n id name   - rename

  !r           - change password
  !quit, !exit - exit (Ctrl+d also)

  other_text   - used as name - start editing existing record.
                 if prefixed with '+' - create if not exists.
`
	STORAGE_FN = "data.db"
)

type Data struct {
	gorm.Model
	Name string
	Text string
}

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
	err = c.Run()
	if err != nil {
		return "", err
	}

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

func main() {

	defer fmt.Println("Bye!")

	fmt.Printf("")

	password := ""
	if p, err := askPass("Password?: "); err != nil {
		panic(err)
	} else {
		password = p
	}

	db, err := gorm.Open("sqlite3", "data.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	p := "PRAGMA key = '" + password + "';"
	err = db.Exec(p).Error
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&Data{}).Error
	if err != nil {
		panic(err)
	}

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

			var dat Data

			err = db.Where("name = ?", name).First(&dat).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound && plus {
					dat = Data{Name: name}
					err = db.Create(&dat).Error
					if err != nil {
						fmt.Println("error: " + err.Error())
						continue
					}
				} else {

					fmt.Println("error: " + err.Error())
					continue
				}
			}

			d, err := displayHidden(dat.Text, name)
			if err != nil {
				fmt.Println("error: " + err.Error())
				continue
			}

			err = db.Model(&dat).Update("Text", string(d)).Error
			if err != nil {
				fmt.Println("error: " + err.Error())
				continue
			}

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

		case "!l":
			if len(command_splitted) != 1 {
				fmt.Println("no params")
				continue
			}

			lst2 := make([]*Data, 0)

			{
				lst := make([]*Data, 0)

				err := db.Find(&lst).Error
				if err != nil {
					fmt.Println("error: " + err.Error())
					continue
				}

				for _, i := range lst {
					lst2 = append(lst2, i)
				}
			}

			if len(lst2) > 1 {
				for i := 0; i != len(lst2)-1; i++ {
					for j := i + 1; j != len(lst2); j++ {
						if lst2[i].Name > lst2[j].Name {
							z := lst2[i]
							lst2[i] = lst2[j]
							lst2[j] = z
						}
					}
				}
			}

			l := ""
			for _, i := range lst2 {
				l += fmt.Sprintf("  %3d '%s'\n", i.ID, i.Name)
			}

			err = useLess(l)
			if err != nil {
				fmt.Println("error: " + err.Error())
				continue
			}

		case "!d":
			if len(command_splitted) != 2 {
				fmt.Println("id required")
				continue
			}

			err = db.Where("id = ?", command_splitted[1]).Delete(&Data{}).Error
			if err != nil {
				fmt.Println(err)
				continue
			}

		case "!n":
			if len(command_splitted) != 3 {
				fmt.Println("id and name required")
				continue
			}

			err = db.Model(&Data{}).Where("id = ?", command_splitted[1]).Update("Name", command_splitted[2]).Error
			if err != nil {
				fmt.Println("error: " + err.Error())
				continue
			}

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

			p := "PRAGMA rekey = '" + password1 + "';"
			err = db.Exec(p).Error
			if err != nil {
				fmt.Println(err)
				continue
			}

		case "!exit":
			fallthrough
		case "!quit":
			break loo
		}
	}

}
