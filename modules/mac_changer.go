package modules

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/bettercap/bettercap/core"
	"github.com/bettercap/bettercap/log"
	bnet "github.com/bettercap/bettercap/network"
	"github.com/bettercap/bettercap/session"
)

type MacChanger struct {
	session.SessionModule
	iface       string
	originalMac net.HardwareAddr
	fakeMac     net.HardwareAddr
}

func NewMacChanger(s *session.Session) *MacChanger {
	mc := &MacChanger{
		SessionModule: session.NewSessionModule("mac.changer", s),
	}

	mc.AddParam(session.NewStringParameter("mac.changer.iface",
		session.ParamIfaceName,
		"",
		"Name of the interface to use."))

	mc.AddParam(session.NewStringParameter("mac.changer.address",
		session.ParamRandomMAC,
		"[a-fA-F0-9]{2}:[a-fA-F0-9]{2}:[a-fA-F0-9]{2}:[a-fA-F0-9]{2}:[a-fA-F0-9]{2}:[a-fA-F0-9]{2}",
		"Hardware address to apply to the interface."))

	mc.AddHandler(session.NewModuleHandler("mac.changer on", "",
		"Start mac changer module.",
		func(args []string) error {
			return mc.Start()
		}))

	mc.AddHandler(session.NewModuleHandler("mac.changer off", "",
		"Stop mac changer module and restore original mac address.",
		func(args []string) error {
			return mc.Stop()
		}))

	return mc
}

func (mc *MacChanger) Name() string {
	return "mac.changer"
}

func (mc *MacChanger) Description() string {
	return "Change active interface mac address."
}

func (mc *MacChanger) Author() string {
	return "Simone Margaritelli <evilsocket@protonmail.com>"
}

func (mc *MacChanger) Configure() (err error) {
	var changeTo string

	if mc.originalMac != nil {
		return errors.New("mac.changer has already been configured, you will need to turn it off to re-configure")
	}

	if err, mc.iface = mc.StringParam("mac.changer.iface"); err != nil {
		return err
	}

	if err, changeTo = mc.StringParam("mac.changer.address"); err != nil {
		return err
	}

	changeTo = bnet.NormalizeMac(changeTo)
	if mc.fakeMac, err = net.ParseMAC(changeTo); err != nil {
		return err
	}

	mc.originalMac = mc.Session.Interface.HW

	return nil
}

func (mc *MacChanger) setMac(mac net.HardwareAddr) error {
	os := runtime.GOOS
	args := []string{}

	if strings.Contains(os, "bsd") || os == "darwin" {
		args = []string{mc.iface, "ether", mac.String()}
	} else if os == "linux" || os == "android" {
		args = []string{mc.iface, "hw", "ether", mac.String()}
	} else {
		return fmt.Errorf("OS %s is not supported by mac.changer module.", os)
	}

	_, err := core.Exec("ifconfig", args)
	if err == nil {
		mc.Session.Interface.HW = mac
	}

	return err
}

func (mc *MacChanger) Start() error {
	if err := mc.Configure(); err != nil {
		return err
	} else if err := mc.setMac(mc.fakeMac); err != nil {
		return err
	}

	return mc.SetRunning(true, func() {
		log.Info("Interface mac address set to %s", core.Bold(mc.fakeMac.String()))
	})
}

func (mc *MacChanger) Stop() error {
	if err := mc.setMac(mc.originalMac); err != nil {
		return err
	}

	// the the module can now be reconfigured
	mc.originalMac = nil

	return mc.SetRunning(false, func() {
		log.Info("Interface mac address restored to %s", core.Bold(mc.originalMac.String()))
	})
}
