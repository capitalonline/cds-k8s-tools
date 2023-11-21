package haproxy

import (
	"cds-k8s-tools/pkg/config"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	ConfigFileName = "reload-lb-system-config"
	ConfigDir      = "/reload-lb/"
	ConfigFmtType  = "ini"
	HaPeriod       = "haproxy.refresh.period"
	PodNodePeriod  = "node.refresh.period"
)

var (
	conf                *config.Conf
	HaRefreshInterval   int
	NodeRefreshInterval int
	HaRefreshCh         chan int
	NodeRefreshCh       chan int
)

func init() {
	conf = config.NewConf(ConfigFileName, ConfigFmtType, false, ConfigDir)
	HaRefreshCh, NodeRefreshCh = make(chan int, 1), make(chan int, 1)
}

func ReloadIntervalFromConf(key string) int {
	return conf.GetKeyInt(key)
}

func AllEventHandle(name string) {
	HaEventHandle(name)
	NodePodEventHandle(name)
}

func NodePodEventHandle(name string) {
	log.Infof("starting event for check nodepod by %s", name)
	if err := UpdateNodePod(); err != nil {
		log.Fatal(err)
	}
	log.Infof("ended event for check nodepod by %s", name)

	// check NodePod Period Second
	NodeRefreshKey := fmt.Sprintf("default.%s", PodNodePeriod)
	newNodePeriod := ReloadIntervalFromConf(NodeRefreshKey)
	if newNodePeriod > 0 && newNodePeriod != NodeRefreshInterval {
		NodeRefreshInterval = newNodePeriod
		log.Infof("updated %s = %d", NodeRefreshKey, newNodePeriod)
		NodeRefreshCh <- newNodePeriod
	} else if newNodePeriod == 0 {
		close(NodeRefreshCh)
	}
}

func HaEventHandle(name string) {
	log.Infof("starting event for check haproxy instance by %s", name)
	if err := UpdateHaproxyInstance(); err != nil {
		log.Fatal(err)
	}
	log.Infof("ended event for check haproxy instance by %s", name)

	// check Haproxy Period Second
	HaRefreshKey := fmt.Sprintf("default.%s", HaPeriod)
	newHaPeriod := ReloadIntervalFromConf(HaRefreshKey)
	if newHaPeriod > 0 && newHaPeriod != HaRefreshInterval {
		HaRefreshInterval = newHaPeriod
		log.Infof("updated %s = %d", HaRefreshKey, newHaPeriod)
		HaRefreshCh <- newHaPeriod
	} else if newHaPeriod == 0 {
		close(HaRefreshCh)
	}

}

func CheckHaProxyInstance(wg *sync.WaitGroup) {

	defer wg.Done()
	newTimer := time.NewTicker(time.Duration(HaRefreshInterval) * time.Second)

	for {
		select {
		case i, ok := <-HaRefreshCh:
			if !ok || i == 0 {
				log.Infof("ending the check haproxy timer")
				newTimer.Stop()
				return
			}
			if i > 0 {
				log.Infof("updating the timer new HaProxyInstance interval %d", i)
				HaRefreshInterval = i
				newTimer.Reset(time.Duration(i) * time.Second)
			}
		case <-newTimer.C:
			HaEventHandle("timer")
		}
	}
}

func CheckNodePod(wg *sync.WaitGroup) {

	defer wg.Done()
	newTimer := time.NewTicker(time.Duration(NodeRefreshInterval) * time.Second)

	for {
		select {
		case i, ok := <-NodeRefreshCh:
			if !ok || i == 0 {
				log.Infof("ending the check nodepod timer")
				newTimer.Stop()
				return
			}
			if i > 0 {
				log.Infof("updating the timer new NodePod interval for %d", i)
				NodeRefreshInterval = i
				newTimer.Reset(time.Duration(i) * time.Second)
			}
		case <-newTimer.C:
			NodePodEventHandle("timer")
		}
	}
}

func Run() {

	HaEventHandle(ConfigFileName)
	NodePodEventHandle(ConfigFileName)

	conf.OnConfChange(AllEventHandle)
	conf.WatchConf()
	wg := new(sync.WaitGroup)
	wg.Add(2)

	go CheckHaProxyInstance(wg)
	go CheckNodePod(wg)

	wg.Wait()
}
