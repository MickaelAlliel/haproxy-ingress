/*
Copyright 2019 The HAProxy Ingress Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package annotations

import (
	"fmt"
	"regexp"

	ingtypes "github.com/jcmoraisjr/haproxy-ingress/pkg/converters/ingress/types"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/utils"
)

func (c *updater) buildGlobalProc(d *globalData) {
	balance := d.mapper.Get(ingtypes.GlobalNbprocBalance).Int()
	if balance < 1 {
		c.logger.Warn("invalid value of nbproc-balance configmap option (%v), using 1", balance)
		balance = 1
	}
	if balance > 1 {
		// need to visit (at least) statistics and healthz bindings as well
		// as admin socket before using more than one balance backend
		c.logger.Warn("nbproc-balance configmap option (%v) greater than 1 is not yet supported, using 1", balance)
		balance = 1
	}
	ssl := d.mapper.Get(ingtypes.GlobalNbprocSSL).Int()
	if ssl < 0 {
		c.logger.Warn("invalid value of nbproc-ssl configmap option (%v), using 0", ssl)
		ssl = 0
	}
	procs := balance + ssl
	threads := d.mapper.Get(ingtypes.GlobalNbthread).Int()
	if threads < 1 {
		c.logger.Warn("invalid value of nbthread configmap option (%v), using 1", threads)
		threads = 1
	}
	bindprocBalance := "1"
	if balance > 1 {
		bindprocBalance = fmt.Sprintf("1-%v", balance)
	}
	bindprocSSL := ""
	if ssl == 0 {
		bindprocSSL = bindprocBalance
	} else if ssl == 1 {
		bindprocSSL = fmt.Sprintf("%v", balance+1)
	} else if ssl > 1 {
		bindprocSSL = fmt.Sprintf("%v-%v", balance+1, procs)
	}
	cpumap := ""
	if threads > 1 {
		if procs == 1 {
			cpumap = fmt.Sprintf("auto:1/1-%v 0-%v", threads, threads-1)
		}
	} else if procs > 1 {
		cpumap = fmt.Sprintf("auto:1-%v 0-%v", procs, procs-1)
	}
	d.global.Procs.Nbproc = procs
	d.global.Procs.Nbthread = threads
	d.global.Procs.NbprocBalance = balance
	d.global.Procs.NbprocSSL = ssl
	d.global.Procs.BindprocBalance = bindprocBalance
	d.global.Procs.BindprocSSL = bindprocSSL
	d.global.Procs.CPUMap = cpumap
}

func (c *updater) buildGlobalSyslog(d *globalData) {
	d.global.Syslog.Endpoint = d.mapper.Get(ingtypes.GlobalSyslogEndpoint).Value
	d.global.Syslog.Format = d.mapper.Get(ingtypes.GlobalSyslogFormat).Value
	d.global.Syslog.HTTPLogFormat = d.mapper.Get(ingtypes.GlobalHTTPLogFormat).Value
	d.global.Syslog.HTTPSLogFormat = d.mapper.Get(ingtypes.GlobalHTTPSLogFormat).Value
	d.global.Syslog.Tag = d.mapper.Get(ingtypes.GlobalSyslogTag).Value
	d.global.Syslog.TCPLogFormat = d.mapper.Get(ingtypes.GlobalTCPLogFormat).Value
}

func (c *updater) buildGlobalTimeout(d *globalData) {
	d.global.Timeout.Client = c.validateTime(d.mapper.Get(ingtypes.HostTimeoutClient))
	d.global.Timeout.ClientFin = c.validateTime(d.mapper.Get(ingtypes.HostTimeoutClientFin))
	d.global.Timeout.Connect = c.validateTime(d.mapper.Get(ingtypes.BackTimeoutConnect))
	d.global.Timeout.HTTPRequest = c.validateTime(d.mapper.Get(ingtypes.BackTimeoutHTTPRequest))
	d.global.Timeout.KeepAlive = c.validateTime(d.mapper.Get(ingtypes.BackTimeoutKeepAlive))
	d.global.Timeout.Queue = c.validateTime(d.mapper.Get(ingtypes.BackTimeoutQueue))
	d.global.Timeout.Server = c.validateTime(d.mapper.Get(ingtypes.BackTimeoutServer))
	d.global.Timeout.ServerFin = c.validateTime(d.mapper.Get(ingtypes.BackTimeoutServerFin))
	d.global.Timeout.Stop = c.validateTime(d.mapper.Get(ingtypes.GlobalTimeoutStop))
	d.global.Timeout.Tunnel = c.validateTime(d.mapper.Get(ingtypes.BackTimeoutTunnel))
}

func (c *updater) buildGlobalSSL(d *globalData) {
	d.global.SSL.Ciphers = d.mapper.Get(ingtypes.GlobalSSLCiphers).Value
	d.global.SSL.Options = d.mapper.Get(ingtypes.GlobalSSLOptions).Value
	if sslDHParam := d.mapper.Get(ingtypes.GlobalSSLDHParam).Value; sslDHParam != "" {
		if dhFile, err := c.cache.GetDHSecretPath(sslDHParam); err == nil {
			d.global.SSL.DHParam.Filename = dhFile.Filename
		} else {
			c.logger.Error("error reading DH params: %v", err)
		}
	}
	d.global.SSL.DHParam.DefaultMaxSize = d.mapper.Get(ingtypes.GlobalSSLDHDefaultMaxSize).Int()
	d.global.SSL.Engine = d.mapper.Get(ingtypes.GlobalSSLEngine).Value
	d.global.SSL.ModeAsync = d.mapper.Get(ingtypes.GlobalSSLModeAsync).Bool()
	d.global.SSL.HeadersPrefix = d.mapper.Get(ingtypes.GlobalSSLHeadersPrefix).Value
}

func (c *updater) buildGlobalModSecurity(d *globalData) {
	d.global.ModSecurity.Endpoints = utils.Split(d.mapper.Get(ingtypes.GlobalModsecurityEndpoints).Value, ",")
	d.global.ModSecurity.Timeout.Hello = c.validateTime(d.mapper.Get(ingtypes.GlobalModsecurityTimeoutHello))
	d.global.ModSecurity.Timeout.Idle = c.validateTime(d.mapper.Get(ingtypes.GlobalModsecurityTimeoutIdle))
	d.global.ModSecurity.Timeout.Processing = c.validateTime(d.mapper.Get(ingtypes.GlobalModsecurityTimeoutProcessing))
}

var (
	forwardRegex = regexp.MustCompile(`^(add|ignore|ifmissing)$`)
)

func (c *updater) buildGlobalForwardFor(d *globalData) {
	if forwardFor := d.mapper.Get(ingtypes.GlobalForwardfor).Value; forwardRegex.MatchString(forwardFor) {
		d.global.ForwardFor = forwardFor
	} else {
		if forwardFor != "" {
			c.logger.Warn("Invalid forwardfor value option on configmap: '%s'. Using 'add' instead", forwardFor)
		}
		d.global.ForwardFor = "add"
	}
}

func (c *updater) buildGlobalCustomConfig(d *globalData) {
	d.global.CustomConfig = utils.LineToSlice(d.mapper.Get(ingtypes.GlobalConfigGlobal).Value)
	d.global.CustomDefaults = utils.LineToSlice(d.mapper.Get(ingtypes.GlobalConfigDefaults).Value)
	d.global.CustomFrontend = utils.LineToSlice(d.mapper.Get(ingtypes.GlobalConfigFrontend).Value)
}
