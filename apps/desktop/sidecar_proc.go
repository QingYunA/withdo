package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// proc is one run of the sidecar process.
type proc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	exited chan struct{}
}

func (p *proc) kill() {
	_ = p.stdin.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

type outcome struct {
	endpoint *Endpoint // set when the process reached its ready line
	reason   string
}

// parseReadyLine reads the line `adhd-core serve` prints once it listens.
func parseReadyLine(line string) (Endpoint, bool) {
	var ep Endpoint
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &ep); err != nil {
		return Endpoint{}, false
	}
	return ep, ep.Port > 0 && ep.Token != ""
}

// runOnce starts the process and returns when it has exited.
func (s *Sidecar) runOnce(previous *Endpoint) outcome {
	cmd := exec.Command(s.cfg.Path, s.cfg.Args...)
	cmd.Env = environ(s.cfg.Env, restartEnv(previous))
	// stdin stays open: `adhd-core serve` exits when it closes, which also covers a crash of ours.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return outcome{reason: fmt.Sprintf("cannot start adhd-core: %v", err)}
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return outcome{reason: fmt.Sprintf("cannot start adhd-core: %v", err)}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return outcome{reason: fmt.Sprintf("cannot start adhd-core: %v", err)}
	}
	if err := cmd.Start(); err != nil {
		return outcome{reason: fmt.Sprintf("cannot start adhd-core: %v", err)}
	}
	p := &proc{cmd: cmd, stdin: stdin, exited: make(chan struct{})}
	if !s.setCurrent(p) {
		p.kill()
		_ = cmd.Wait()
		return outcome{reason: errStopped.Error()}
	}

	ready := make(chan Endpoint, 1)
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		sent := false
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			if ep, ok := parseReadyLine(sc.Text()); ok && !sent {
				sent = true
				ready <- ep
			}
		}
	}()
	go func() {
		defer readers.Done()
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			s.cfg.Logf("[adhd-core] %s", sc.Text())
		}
	}()
	exit := make(chan error, 1)
	go func() {
		readers.Wait() // all reads must finish before Wait closes the pipes
		exit <- cmd.Wait()
		close(p.exited)
	}()

	timer := time.NewTimer(s.cfg.ReadyTimeout)
	defer timer.Stop()
	var got *Endpoint
	select {
	case ep := <-ready:
		got = &ep
		if s.publish(stateReady, ep, "") {
			err = <-exit
		} else {
			p.kill()
			err = <-exit
		}
		return outcome{endpoint: got, reason: exitReason(err)}
	case err = <-exit:
		select { // it may have printed the line just before it died
		case ep := <-ready:
			got = &ep
		default:
		}
		return outcome{endpoint: got, reason: exitReason(err)}
	case <-timer.C:
		p.kill()
		<-exit
		return outcome{reason: fmt.Sprintf("adhd-core did not report a port within %s", s.cfg.ReadyTimeout)}
	}
}

func exitReason(err error) string {
	if err == nil {
		return "adhd-core exited"
	}
	return "adhd-core exited: " + err.Error()
}
