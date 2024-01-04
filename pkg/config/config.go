package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

type Conf struct {
	Name      string
	Type      string
	IsWatched bool
	*viper.Viper
}

func NewConf(fName string, fType string, needWatch bool, fPath ...string) *Conf {
	if fName == "" || fType == "" || fPath == nil {
		panic(fmt.Errorf("params fName:(%s) or fType:(%s) or fPath:(%s) blank", fName, fType, fPath))
	}
	log.Infof("starting read new %s conf(%s) from %s", fType, fName, fPath)
	v := viper.New()
	v.SetConfigName(fName)
	v.SetConfigType(fType)
	for _, path := range fPath {
		v.AddConfigPath(path)
	}
	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}
	log.Infof("ended read new %s conf(%s) from(%s)", fType, fName, fPath)
	return &Conf{
		Name:      fName,
		Type:      fType,
		IsWatched: needWatch,
		Viper:     v,
	}
}

func (c *Conf) OnConfChange(fc func(name string)) {
	c.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("File(%s) changed at %s.", e.Name, time.Now().String())
		fc(e.Name)
	})
}

func (c *Conf) WatchConf() {
	c.WatchConfig()
}
func (c *Conf) HasKey(head, k string) bool {
	return c.IsSet(head + k)
}

func (c *Conf) GetKeyString(head, k string) string {
	return c.GetString(head + k)
}

func (c *Conf) GetKeyInt(head, k string) int {
	return c.GetInt(head + k)
}
