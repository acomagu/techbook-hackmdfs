package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/acomagu/techbook-hackmdfs/go-hackmd"
	"github.com/graarh/golang-socketio/transport"
)

const sessionID = "xxxx"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "specify command")
		os.Exit(1)
	}

	ctx := context.Background()

	hmd := hackmd.NewClient(sessionID, nil)

	if err := func() error {
		cmd := os.Args[1]
		switch cmd {
		case "history":
			entries, err := hmd.GetHistory(ctx)
			if err != nil {
				return err
			}

			for _, entry := range entries {
				fmt.Printf("%s\t%v\t%s\n", entry.ID, time.Unix(int64(entry.Time/1000), 0), entry.Text)
			}

		case "write":
			if len(os.Args) < 3 {
				return fmt.Errorf("please specify arguments: write <note_id> <content>")
			}

			noteID := os.Args[2]

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://hackmd.io/realtime-1/socket.io/?noteId=%s&transport=polling", noteID), nil)
			if err != nil {
				return err
			}
			req.Header.Add("cookie", "connect.sid=xxxx")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			bt := make([]byte, 1)
			for {
				if _, err := resp.Body.Read(bt); err == io.EOF {
					return fmt.Errorf("unexpected EOF")
				} else if err != nil {
					return err
				}

				if _, err := strconv.Atoi(string(bt)); err != nil && bt[0] != ':' {
					break
				}
			}

			// presp := new(struct {
			// 	SID string
			// })
			// if err := json.NewDecoder(io.MultiReader(bytes.NewBuffer(bt), resp.Body)).Decode(presp); err != nil {
			// 	return err
			// }
			//
			// conf, err := websocket.NewConfig(
			// 	fmt.Sprintf("wss://hackmd.io/realtime-1/socket.io/?transport=websocket&EIO=3&noteId=%s&sid=%s", noteID, presp.SID),
			// 	"https://hackmd.io/",
			// )
			// if err != nil {
			// 	return err
			// }
			// conf.Header.Add("cookie", "connect.sid=xxxx")
			// c, err := websocket.DialConfig(conf)
			// if err != nil {
			// 	return err
			// }
			//
			// go func() {
			// 	buf := make([]byte, 4096)
			// 	for {
			// 		n, err := c.Read(buf)
			// 		if err != nil {
			// 			fmt.Fprintln(os.Stderr, err)
			// 			break
			// 		}
			// 		fmt.Println(string(buf[:n]))
			// 	}
			// }()
			//
			// c.Write([]byte("2probe"))
			// time.Sleep(time.Second)
			// c.Write([]byte("5"))
			// c.Write([]byte("2"))
			// time.Sleep(time.Second)
			// c.Write([]byte(`42["user status",{"idle":false,"type":"xs"}]`))

			c, err := gosocketio.Dial(
				fmt.Sprintf("wss://hackmd.io/realtime-1/socket.io/?transport=websocket&EIO=3&noteId=%s&sid=%s", noteID, presp.SID),
				transport.GetDefaultWebsocketTransport(),
			)

			fmt.Println(c.On(gosocketio.OnConnection, func(h *gosocketio.Channel) {
				fmt.Println("connected")

				fmt.Println(c.On("check", func(h *gosocketio.Channel, args interface{}) {
					fmt.Println(args)
				}))
				fmt.Println(c.On(gosocketio.OnError, func(h *gosocketio.Channel, args interface{}) {
					fmt.Println("error", args)
				}))

				fmt.Println(h.Emit("user status", []interface{}{
					map[string]interface{}{
						"idle": false,
						"type": "xs",
					},
				}))

				fmt.Println(h.Emit("operation", []interface{}{
					1,
					[]interface{}{19, "h", 2},
					map[string]interface{}{
						"ranges": []interface{}{
							map[string]interface{}{
								"anchor": 20,
								"head":   20,
							},
						},
					},
				}))
				h.Emit("disconnect", nil)
				h.Close()
			}))
			fmt.Println(c.On("user status", func(h *gosocketio.Channel, args interface{}) {
				fmt.Println("user status", args)
			}))
			fmt.Println(c.On("disconnect", func(h *gosocketio.Channel, args interface{}) {
				fmt.Println("disconnect", args)
			}))
			fmt.Println(c.On(gosocketio.OnDisconnection, func(h *gosocketio.Channel, args interface{}) {
				fmt.Println("disconnection", args)
			}))
			fmt.Println(c.On("connect", func(h *gosocketio.Channel) {
				fmt.Println("connected")

				fmt.Println(c.On("check", func(h *gosocketio.Channel, args interface{}) {
					fmt.Println(args)
				}))
				fmt.Println(c.On(gosocketio.OnError, func(h *gosocketio.Channel, args interface{}) {
					fmt.Println("error", args)
				}))

				fmt.Println(h.Emit("user status", []interface{}{
					map[string]interface{}{
						"idle": false,
						"type": "xs",
					},
				}))

				fmt.Println(h.Emit("operation", []interface{}{
					1,
					[]interface{}{19, "h", 2},
					map[string]interface{}{
						"ranges": []interface{}{
							map[string]interface{}{
								"anchor": 20,
								"head":   20,
							},
						},
					},
				}))
				h.Emit("disconnect", nil)
				h.Close()
			}))

			<-make(chan struct{})
		}

		return nil
	}(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
