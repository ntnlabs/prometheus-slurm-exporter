/* Copyright 2020 Victor Penso

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>. */

package main

import (
        "io/ioutil"
        "os/exec"
        "log"
        "strings"
        "regexp"
        "github.com/prometheus/client_golang/prometheus"
)

func AccountsData() []byte {
        cmd := exec.Command("squeue", "-h", "-o '%A|%a|%T'")
        stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	out, _ := ioutil.ReadAll(stdout)
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	return out
}

type JobMetrics struct {
        cancelled float64
        completed float64
        pending float64
        running float64
        suspended float64
}

func ParseAccountsMetrics(input []byte) map[string]*JobMetrics {
        accounts := make(map[string]*JobMetrics)
        lines := strings.Split(string(input), "\n")
        for _, line := range lines {
                if strings.Contains(line,"|") {
                        account := strings.Split(line,"|")[1]
                        _,key := accounts[account]
                        if !key {
                                accounts[account] = &JobMetrics{0,0,0,0,0}
                        }
                        state := strings.Split(line,"|")[2]
                        state = strings.ToLower(state)
                        cancelled := regexp.MustCompile(`^cancelled`)
                        completed := regexp.MustCompile(`^completed`)
                        pending := regexp.MustCompile(`^pending`)
                        running := regexp.MustCompile(`^running`)
                        suspended := regexp.MustCompile(`^suspended`)
                        switch {
                        case cancelled.MatchString(state) == true:
                                accounts[account].cancelled++
                        case completed.MatchString(state) == true:
                                accounts[account].completed++
                        case pending.MatchString(state) == true:
                                accounts[account].pending++
                        case running.MatchString(state) == true:
                                accounts[account].running++
                        case suspended.MatchString(state) == true:
                                accounts[account].suspended++
                        }
                }
        }
        return accounts
}

type AccountsCollector struct {
        cancelled *prometheus.Desc
        completed *prometheus.Desc
        pending *prometheus.Desc
        running *prometheus.Desc
        suspended *prometheus.Desc
}

func NewAccountsCollector() *AccountsCollector {
        labels := []string{"account"}
        return &AccountsCollector{
                cancelled: prometheus.NewDesc("slurm_account_jobs_cancelled", "Cancelled jobs for account", labels, nil),
                completed: prometheus.NewDesc("slurm_account_jobs_completed", "Completed jobs for account", labels, nil),
                running: prometheus.NewDesc("slurm_account_jobs_running", "Running jobs for account", labels, nil),
                pending: prometheus.NewDesc("slurm_account_jobs_pending", "Pending jobs for account", labels, nil),
                suspended: prometheus.NewDesc("slurm_account_jobs_suspended", "Suspended jobs for account", labels, nil),
        }
}

func (ac *AccountsCollector) Describe(ch chan<- *prometheus.Desc) {
        ch <- ac.cancelled
        ch <- ac.completed
        ch <- ac.pending
        ch <- ac.running
        ch <- ac.suspended
}

func (ac *AccountsCollector) Collect(ch chan<- prometheus.Metric) {
        am := ParseAccountsMetrics(AccountsData())
        for a := range am {
                ch <- prometheus.MustNewConstMetric(ac.cancelled, prometheus.GaugeValue, am[a].cancelled, a)
                ch <- prometheus.MustNewConstMetric(ac.completed, prometheus.GaugeValue, am[a].completed, a)
                ch <- prometheus.MustNewConstMetric(ac.pending, prometheus.GaugeValue, am[a].pending, a)
                ch <- prometheus.MustNewConstMetric(ac.running, prometheus.GaugeValue, am[a].running, a)
                ch <- prometheus.MustNewConstMetric(ac.suspended, prometheus.GaugeValue, am[a].suspended, a)
        }
}