package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gobak/fileutils"
	"gobak/level"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//Errors in the configuration file
var (
	ErrFolderBackupNotExists = errors.New("Config: Folder for backup not found")
	ErrConfigLevel           = errors.New("Config: levels not found")
	ErrNbackupNotExists      = errors.New("Config: file Nbackup destination not exists")
	ErrGfixNotExists         = errors.New("Config: file gfix  destination not exists")
	ErrPhysicalNotExists     = errors.New("Config: Physicalpathdb destination not exists")
	ErrAliasDBNotExists      = errors.New("Config: Alias DB is empty")
)

var cfg *Config

//A Config it contains the application settings from a file config.json
type Config struct {
	PathToNbackup      string `json:"PathToNbackup"`
	PathToBackupFolder string `json:"PathToBackupFolder"`
	DirectIO           bool   `json:"DirectIO"`
	AliasDb            string `json:"AliasDb"`
	Password           string `json:"Password"`
	User               string `json:"User"`
	EmailFrom          string `json:"EmailFrom"`
	EmailTo            string `json:"EmailTo"`
	SMTPServer         string `json:"SmtpServer"`
	Pathtogfix         string `json:"Pathtogfix"`
	Physicalpathdb     string `json:"Physicalpathdb"`
	NameBase           string `json:"NameBase"`
	TimeMsec           int    `json:"TimeMlsc"`

	Levels []struct {
		Level int    `json:"level"`
		Tick  string `json:"tick"`
		Check bool   `json:"check"`
	} `json:"levels"`

	file         string
	LevelsConfig *level.Levels
}

//Current returns a *Config each time one and the same or or will be creating it
func Current() *Config {
	if cfg == nil {
		fileconfig := "config.json"
		if _, e := os.Stat(fileconfig); e != nil && os.IsNotExist(e) {
			fileconfig = filepath.Join(filepath.Dir(os.Args[0]), "config.json")
		}
		if cfg == nil {
			var err error
			cfg, err = loadConfig(fileconfig)
			if err != nil {
				panic(err)
			}
			cfg.file = fileconfig
		}
	}
	return cfg
}

func valueDefault(val, def interface{}) interface{} {
	if val == nil {
		return def
	}
	return val
}

func lookupEnv(key string) (result string) {
	result, _ = os.LookupEnv(key)
	return result
}

func loadConfig(filename string) (result *Config, err error) {
	file, e := ioutil.ReadFile(filename)
	if e != nil {
		return nil, e
	}
	result = &Config{}
	var res map[string]interface{}
	if e = json.Unmarshal(file, &res); e != nil {
		return nil, e
	}
	result.PathToNbackup = res["PathToNbackup"].(string)
	result.PathToBackupFolder = filepath.Clean(valueDefault(res["PathToBackupFolder"], "").(string))
	result.AliasDb = strings.TrimSpace(valueDefault(res["AliasDb"], "").(string))
	result.Physicalpathdb = valueDefault(res["Physicalpathdb"], "").(string)
	result.Password = valueDefault(res["Password"], lookupEnv("ISC_PASSWORD")).(string)
	result.User = valueDefault(res["User"], lookupEnv("ISC_USER")).(string)
	result.EmailFrom = valueDefault(res["EmailFrom"], "").(string)
	result.EmailTo = valueDefault(res["EmailTo"], "").(string)
	result.SMTPServer = valueDefault(res["SmtpServer"], "").(string)
	result.Pathtogfix = valueDefault(res["Pathtogfix"], "").(string)
	result.NameBase = valueDefault(res["NameBase"], "").(string)
	result.TimeMsec = int(valueDefault(res["TimeMlsc"], 10000).(float64))
	result.DirectIO = valueDefault(res["DirectIO"], false).(bool)
	result.LevelsConfig = level.NewList()

	for _, p := range res["levels"].([]interface{}) {
		litem := p.(map[string]interface{})
		cfg, err := result.LevelsConfig.Add(
			level.NewLevel(int(litem["level"].(float64))),
			level.NewTick(litem["tick"].(string)))

		if err != nil {
			return nil, err
		}
		if b, ok := litem["check"]; ok {
			cfg.Check = b.(bool)
		}
	}
	return result, nil
}

//Check config file
func (c *Config) Check() error {
	if !fileutils.Exists(c.PathToNbackup) {
		return ErrNbackupNotExists
	}
	if !fileutils.Exists(c.Pathtogfix) {
		return ErrGfixNotExists
	}
	if !fileutils.Exists(c.Physicalpathdb) {
		return ErrPhysicalNotExists
	}
	if !fileutils.Exists(c.PathToBackupFolder) {
		return ErrFolderBackupNotExists
	}
	if f, e := os.Stat(c.PathToBackupFolder); e != nil && (os.IsNotExist(e) || !f.IsDir()) {
		return ErrFolderBackupNotExists
	}
	if c.LevelsConfig.Count() == 0 {
		return ErrConfigLevel
	}
	if c.AliasDb == "" {
		return ErrAliasDBNotExists
	}
	return nil
}

//String it Stringer
func (c *Config) String() string {
	var buffer bytes.Buffer
	s := fmt.Sprintf("Config: %s\n", c.file) +
		fmt.Sprintf("Database: name: %q, alias %q, path %q\n", c.NameBase, c.AliasDb, c.Physicalpathdb) +
		fmt.Sprintf("Backup Folder: %s\n", c.PathToBackupFolder) +
		fmt.Sprintf("SMTP Server: %s\n", c.SMTPServer) +
		fmt.Sprintf("Schedule backup:%s\n", c.LevelsConfig.Schedule())
	if _, err := buffer.WriteString(s); err != nil {
		panic(err)
	}
	return buffer.String()
}
