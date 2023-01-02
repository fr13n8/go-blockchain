package main

import (
	"context"
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	pb "github.com/fr13n8/go-blockchain/gen/node"
	"github.com/fr13n8/go-blockchain/network/discovery"
	"github.com/fr13n8/go-blockchain/server"
	"github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/url"
	"strconv"
)

type Logs struct {
	logsChannel chan string
}

var (
	srv            = server.NewServer()
	logsWidget     = widget.NewMultiLineEntry()
	nodeClient     pb.NodeServiceClient
	peers          = make([]string, 0)
	discoveryPeers = make(chan []string)
)

func (l *Logs) Write(data []byte) (n int, err error) {
	l.logsChannel <- string(data)
	return len(data), nil
}

var logs = Logs{
	logsChannel: make(chan string),
}

func listenNode(bootNodes []multiaddr.Multiaddr, exit <-chan struct{}) {
	nodeAddr := srv.NodeServer.Run()
	conn, err := grpc.Dial(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[APP] Error while connecting to node gateway: %s", err.Error())
		return
	}
	nodeClient = pb.NewNodeServiceClient(conn)
	srv.PeerDiscovery.Run(bootNodes, discoveryPeers)
	<-exit
	srv.NodeServer.ShutdownGracefully()
	srv.PeerDiscovery.ShutdownGracefully()
}

func listenBLockExplorer(port int, exit <-chan struct{}) {
	srv.BlockExplorer.Run(port)
	<-exit
	log.Printf("[BLOCK-EXPLORER] Stop block-explorer listen on port %d", port)
	srv.BlockExplorer.ShutdownGracefully()
}

func main() {
	myApp := app.New()
	w := myApp.NewWindow("Node")

	listening := false
	bootNodes := widget.NewEntry()
	bootNodes.SetPlaceHolder("Set boot node or leave empty if you want to be a boot node")

	var toggleNodeListenButton *widget.Button
	stopNode := make(chan struct{})
	toggleNodeListenButton = widget.NewButton("Start node listen", func() {
		nodes := make([]multiaddr.Multiaddr, 0)
		if listening {
			toggleNodeListenButton.SetText("Start node listen")
			stopNode <- struct{}{}
		} else {
			bNode := bootNodes.Text
			if bNode == "" {
				nodes = append(nodes, []multiaddr.Multiaddr{}...)
			} else {
				p, err := discovery.StringsToAddrs([]string{bNode})
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				nodes = append(nodes, p...)
			}
			go listenNode(nodes, stopNode)
			toggleNodeListenButton.SetText("Stop node listen")
		}
		listening = !listening
	})
	nodeListen := container.NewGridWithColumns(2, bootNodes, toggleNodeListenButton)

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
			p, err := strconv.ParseInt(port, 10, 16)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			go listenBLockExplorer(int(p), stopBlockExplorerRunning)
			toggleBlockExplorerRunningRunButton.SetText("Stop block explorer ui")
		}
		blockExplorerRunning = !blockExplorerRunning
	})
	blockExplorerUi := container.NewGridWithColumns(2, blockExplorerRunningListenPort, toggleBlockExplorerRunningRunButton)

	peersList := widget.NewList(
		func() int {
			return len(peers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(peers[i])
		})

	logsWidget.Disable()
	logsWidget.TextStyle.Monospace = true
	go func(peers *[]string, list *widget.List) {
		for {
			select {
			case newPeers := <-discoveryPeers:
				*peers = newPeers
				list.Refresh()
			}
		}
	}(&peers, peersList)

	logsCard := widget.NewCard("Logs", "", logsWidget)
	listCard := widget.NewCard("Nodes", "", peersList)
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

	openBlockExplorer := container.NewGridWithColumns(1, blockExplorerHyperLink)

	vBox := container.NewVBox(
		nodeListen,
		blockExplorerUi,
		openBlockExplorer,
		minerAddressEntry,
		startMining,
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
