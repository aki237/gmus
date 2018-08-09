package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type status struct {
	Playing  bool
	File     string
	Duration int
	Position int
	Artist   string
	Album    string
	Title    string
	Date     string
}

type cmusSocket struct {
	conn net.Conn
}

func newCmusSocket() (*cmusSocket, error) {
	sockpath := os.Getenv("CMUS_SOCKET")
	if sockpath == "" {
		sockpath = os.Getenv("XDG_RUNTIME_DIR")
		if sockpath == "" {
			return nil, errors.New("cannot determine the cmus socket path")
		}
		sockpath = filepath.Join(sockpath, "cmus-socket")
	}
	conn, err := net.Dial("unix", sockpath)
	if err != nil {
		return nil, err
	}

	return &cmusSocket{conn: conn}, nil
}

func (c *cmusSocket) VolUp() bool {
	fmt.Fprint(c.conn, "vol +1%\n")
	p := make([]byte, 1)
	c.conn.Read(p)
	if p[0] != '\n' {
		return false
	}
	return true
}

func (c *cmusSocket) VolDown() bool {
	fmt.Fprint(c.conn, "vol -1%\n")
	p := make([]byte, 1)
	c.conn.Read(p)
	if p[0] != '\n' {
		return false
	}
	return true
}

func (c *cmusSocket) Seek(second int) bool {
	fmt.Fprintf(c.conn, "seek %d\n", second)
	p := make([]byte, 1)
	c.conn.Read(p)
	if p[0] != '\n' {
		return false
	}
	return true
}

func (c *cmusSocket) Next() bool {
	fmt.Fprintf(c.conn, "player-next\n")
	p := make([]byte, 1)
	c.conn.Read(p)
	if p[0] != '\n' {
		return false
	}
	return true
}

func (c *cmusSocket) Prev() bool {
	fmt.Fprintf(c.conn, "player-prev\n")
	p := make([]byte, 1)
	c.conn.Read(p)
	if p[0] != '\n' {
		return false
	}
	return true
}

func (c *cmusSocket) TogglePausePlay() bool {
	fmt.Fprintf(c.conn, "player-pause\n")
	p := make([]byte, 1)
	c.conn.Read(p)
	if p[0] != '\n' {
		return false
	}
	return true
}

func (c *cmusSocket) GetStatus() (*status, error) {
	c.conn.Write([]byte("status\n"))
	br := bufio.NewReader(c.conn)
	s := &status{}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimPrefix(line, "set")
		line = strings.TrimPrefix(line, "tag")
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			break
		}

		splits := strings.SplitN(line, " ", 2)
		if len(splits) != 2 {
			continue
		}

		switch splits[0] {
		case "status":
			if splits[1] == "playing" {
				s.Playing = true
			}
		case "file":
			s.File = splits[1]
		case "duration", "position":
			num, err := strconv.Atoi(splits[1])
			if err != nil {
				return nil, err
			}
			if splits[0] == "duration" {
				s.Duration = num
			}
			if splits[0] == "position" {
				s.Position = num
			}
		case "artist":
			s.Artist = splits[1]
		case "album":
			s.Album = splits[1]
		case "title":
			s.Title = splits[1]
		case "date":
			s.Date = splits[1]
		}

	}

	if s.Artist == "" {
		s.Artist = "Unknown Artist"
	}
	if s.Album == "" {
		s.Album = "Unknown Album"
	}
	if s.Title == "" {
		s.Title = filepath.Base(s.File)
	}
	if s.Date == "" {
		s.Date = "Unknown Date"
	}
	return s, nil
}
