package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"fyne.io/fyne/v2/layout"
	block_explorer "github.com/fr13n8/go-blockchain/pkg/block-explorer/server"
	pb "github.com/fr13n8/go-blockchain/pkg/network/node"
	nodeServer "github.com/fr13n8/go-blockchain/pkg/node/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Logs struct {
	logsChannel chan string
}

var (
	logsWidget = widget.NewMultiLineEntry()
	nodeClient pb.NodeServiceClient
)

func (l *Logs) Write(data []byte) (n int, err error) {
	l.logsChannel <- string(data)
	return len(data), nil
}

var logs = Logs{
	logsChannel: make(chan string),
}

func listenNode(port uint16, exit <-chan struct{}) {
	cfg := nodeServer.Config{
		Port:       port,
		ServerName: "Node",
		Host:       "0.0.0.0",
	}
	s := nodeServer.NewServer(&cfg)
	log.Printf("[NODE] Start node listen on port %d", port)
	s.Run()
	conn, err := grpc.Dial("0.0.0.0:"+fmt.Sprintf("%d", port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[NODE] Error while connecting to gateway: %s", err.Error())
	}
	nodeClient = pb.NewNodeServiceClient(conn)
	<-exit
	log.Printf("[NODE] Stop node listen on port %d", port)
	s.ShutdownGracefully()
}

func listenBLockExplorer(port uint16, gateway string, exit <-chan struct{}) {
	cfg := block_explorer.Config{
		Port:       port,
		ServerName: "Wallet",
		Host:       "0.0.0.0",
		Gateway:    gateway,
	}
	s := block_explorer.NewServer(&cfg)
	log.Printf("[BLOCK-EXPLORER] Start block-explorer listen on port %d", port)
	s.Run()
	<-exit
	log.Printf("[WALLET] Stop wallet listen on port %d", port)
	s.ShutdownGracefully()
}

func loadJsonData() [][]string {
	fmt.Println("Loading data from JSON file")

	input, _ := os.ReadFile("nodes.json")
	var data []string
	json.Unmarshal(input, &data)

	nodes := make([][]string, 0, len(data))
	nodes = append(nodes, []string{"host", "port", "status"})
	for _, h := range data {
		status := "active"
		_, err := net.Dial("tcp", h)
		if err != nil {
			status = "inactive"
		}
		trimmed := strings.Split(h, ":")
		node := []string{trimmed[0], trimmed[1], status}
		nodes = append(nodes, node)
	}
	return nodes
}

func saveJsonData(data [][]string) {
	fmt.Println("Saving data to JSON file")
	nodes := make([]string, 0, len(data)-1)

	for i := 1; i <= len(data)-1; i++ {
		h := data[i]
		node := fmt.Sprintf("%s:%s", h[0], h[1])
		nodes = append(nodes, node)
	}

	jsonData, err := json.Marshal(nodes)
	if err != nil {
		log.Fatalf("ERROR: %s", err.Error())
	}

	os.WriteFile("nodes.json", jsonData, 0644)
}

func main() {
	myApp := app.New()
	w := myApp.NewWindow("Node")

	listening := false
	nodeListenPort := widget.NewEntry()
	nodeListenPort.SetPlaceHolder("Set node listen port")
	nodeListenPort.SetText("5050")
	var toggleNodeListenButton *widget.Button
	stopNode := make(chan struct{})
	toggleNodeListenButton = widget.NewButton("Start node listen", func() {
		if nodeListenPort.Text == "" {
			err := errors.New("Please set port")
			dialog.ShowError(err, w)
			return
		}
		if listening {
			toggleNodeListenButton.SetText("Start node listen")
			stopNode <- struct{}{}
		} else {
			port := nodeListenPort.Text
			p, err := strconv.ParseUint(port, 10, 16)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			go listenNode(uint16(p), stopNode)
			toggleNodeListenButton.SetText("Stop node listen")
		}
		listening = !listening
	})
	nodeListen := container.NewGridWithColumns(2, nodeListenPort, toggleNodeListenButton)

	minerAddress := widget.NewEntry()
	minerAddress.SetPlaceHolder("Set miner address")
	minerAddressEntry := container.NewGridWithColumns(1, minerAddress)

	mining := false
	var toggleStartMiningButton *widget.Button
	toggleStartMiningButton = widget.NewButton("Start mining", func() {
		if nodeClient == nil {
			err := errors.New("Please start node listen")
			dialog.ShowError(err, w)
			return
		}
		if mining {
			log.Println("[NODE] Stopping mining...")

			stopMiningRequest := &pb.StopMiningRequest{}
			stopMiningResponse, err := nodeClient.StopMining(context.Background(), stopMiningRequest)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if stopMiningResponse.Status {
				mining = false
				toggleStartMiningButton.SetText("Start mining")
			}

			return
		} else {
			log.Println("[NODE] Starting mining...")

			if minerAddress.Text == "" {
				err := errors.New("Please set miner address")
				dialog.ShowError(err, w)
				return
			}

			startMiningRequest := &pb.StartMiningRequest{
				MinerAddress: minerAddress.Text,
			}

			startMiningResponse, err := nodeClient.StartMining(context.Background(), startMiningRequest)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			if !startMiningResponse.Status {
				log.Printf("[NODE] Error start mining: %v", startMiningResponse.Status)
				dialog.ShowError(errors.New("Error start mining"), w)
				return
			}

			log.Println("[NODE] Mining started")
			toggleStartMiningButton.SetText("Stop mining")
		}
		mining = !mining
	})
	startMining := container.NewGridWithColumns(1, toggleStartMiningButton)

	blockExplorerRunning := false
	blockExplorerRunningListenPort := widget.NewEntry()
	blockExplorerRunningListenPort.SetPlaceHolder("Set block explorer ui port")
	blockExplorerRunningListenPort.SetText("6060")
	var toggleBlockExplorerRunningRunButton *widget.Button
	stopBlockExplorerRunning := make(chan struct{})
	toggleBlockExplorerRunningRunButton = widget.NewButton("Start block explorer ui", func() {
		if blockExplorerRunningListenPort.Text == "" {
			err := errors.New("Please set port")
			dialog.ShowError(err, w)
			return
		}
		if blockExplorerRunning {
			toggleBlockExplorerRunningRunButton.SetText("Start block explorer ui")
			stopBlockExplorerRunning <- struct{}{}
		} else {
			port := blockExplorerRunningListenPort.Text
			p, err := strconv.ParseUint(port, 10, 16)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			nodeGateway := "0.0.0.0:" + nodeListenPort.Text

			go listenBLockExplorer(uint16(p), nodeGateway, stopBlockExplorerRunning)
			toggleBlockExplorerRunningRunButton.SetText("Stop block explorer ui")
		}
		blockExplorerRunning = !blockExplorerRunning
	})
	blockExplorerUi := container.NewGridWithColumns(2, blockExplorerRunningListenPort, toggleBlockExplorerRunningRunButton)

	loadedData := loadJsonData()
	var data = loadedData
	list := widget.NewTable(
		func() (int, int) {
			return len(data), len(data[0])
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("wide content")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(data[i.Row][i.Col])
		})

	add := widget.NewButton("Add node", func() {
		w := myApp.NewWindow("Add new node")

		itemName := widget.NewEntry()

		addData := widget.NewButton("Add", func() {
			trimmed := strings.Split(itemName.Text, ":")
			data = append(data, []string{trimmed[0], trimmed[1], "active"})
			saveJsonData(data)
			w.Close()
		})

		cancel := widget.NewButton("Cancel", func() {
			w.Close()
		})

		w.SetContent(container.New(layout.NewVBoxLayout(), itemName, addData, cancel))
		w.Resize(fyne.NewSize(400, 100))
		w.SetFixedSize(true)
		w.CenterOnScreen()
		w.Show()

	})

	logsWidget.Disable()
	logsWidget.TextStyle.Monospace = true

	logsCard := widget.NewCard("Logs", "", logsWidget)
	listCard := widget.NewCard("Nodes", "", list)
	split := container.NewVSplit(listCard, logsCard)
	split.Offset = 0.4
	w.SetContent(split)

	blockExplorerLink := "http://localhost:" + blockExplorerRunningListenPort.Text + "/"
	blockExplorerUrl, err := url.Parse(blockExplorerLink)
	if err != nil {
		dialog.ShowError(errors.New("invalid url"), w)
		return
	}
	blockExplorerHyperLink := widget.NewHyperlink("Open block explorer", blockExplorerUrl)

	walletLink := "http://localhost:8080" + "/"
	walletUrl, err := url.Parse(walletLink)
	if err != nil {
		dialog.ShowError(errors.New("invalid url"), w)
		return
	}
	walletHyperLink := widget.NewHyperlink("Open wallet website", walletUrl)

	openBlockExplorer := container.NewGridWithColumns(2, blockExplorerHyperLink, walletHyperLink)

	vBox := container.NewVBox(
		nodeListen,
		blockExplorerUi,
		openBlockExplorer,
		minerAddressEntry,
		startMining,
		add,
	)

	panel := container.NewBorder(vBox, nil, nil, nil, split)
	w.SetContent(panel)
	w.Resize(fyne.NewSize(940, 780))
	w.SetFixedSize(false)
	w.ShowAndRun()
}

func init() {
	log.SetOutput(&logs)
	go func() {
		for {
			newLog := <-logs.logsChannel
			logsWidget.SetText(logsWidget.Text + newLog)
		}
	}()
}
