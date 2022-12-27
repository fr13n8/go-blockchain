package main

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"

	walletServer "github.com/fr13n8/go-blockchain/cmd/client/server"
	nodeServer "github.com/fr13n8/go-blockchain/cmd/node/server"

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
	<-exit
	log.Printf("[NODE] Stop node listen on port %d", port)
	s.ShutdownGracefully()
}

func listenWallet(port uint16, gateway string, exit <-chan struct{}) {
	cfg := walletServer.Config{
		Port:       port,
		ServerName: "Wallet",
		Host:       "0.0.0.0",
		Gateway:    gateway,
	}
	s := walletServer.NewServer(&cfg)
	log.Printf("[WALLET] Start wallet listen on port %d", port)
	s.Run()
	<-exit
	log.Printf("[WALLET] Stop wallet listen on port %d", port)
	s.ShutdownGracefully()
}

func main() {
	myApp := app.New()
	w := myApp.NewWindow("Node")

	mining := false
	var toggleStartMiningButton *widget.Button
	toggleStartMiningButton = widget.NewButton("Start mining", func() {
		if mining {
			log.Println("[NODE] Stopping mining...")
			response, err := http.Get("http://localhost:5050/api/mine/stop")
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if response.StatusCode != http.StatusOK {
				log.Printf("[NODE] Error stop mining: %s", response.Status)
				dialog.ShowError(errors.New("Error stop mining"), w)
				return
			}

			toggleStartMiningButton.SetText("Start mining")
		} else {
			log.Println("[NODE] Starting mining...")
			response, err := http.Get("http://localhost:5050/api/mine/start")
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if response.StatusCode != http.StatusOK {
				log.Printf("[NODE] Error start mining: %s", response.Status)
				dialog.ShowError(errors.New("Error start mining"), w)
				return
			}

			toggleStartMiningButton.SetText("Stop mining")
		}
		mining = !mining
	})
	startMining := container.NewGridWithColumns(1, toggleStartMiningButton)

	blockExplorerLink := "http://localhost:5050" + "/"
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
	walletHyperLink := widget.NewHyperlink("Open wallet", walletUrl)

	openBlockExplorer := container.NewGridWithColumns(2, blockExplorerHyperLink, walletHyperLink)

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
	nodeLIsten := container.NewGridWithColumns(2, nodeListenPort, toggleNodeListenButton)

	walletRunning := false
	walletListenPort := widget.NewEntry()
	walletListenPort.SetPlaceHolder("Set wallet ui port")
	walletListenPort.SetText("8080")
	var toggleWalletRunButton *widget.Button
	stopWallet := make(chan struct{})
	toggleWalletRunButton = widget.NewButton("Start wallet ui", func() {
		if walletListenPort.Text == "" {
			err := errors.New("Please set port")
			dialog.ShowError(err, w)
			return
		}
		if walletRunning {
			toggleWalletRunButton.SetText("Start wallet ui")
			stopWallet <- struct{}{}
		} else {
			port := walletListenPort.Text
			p, err := strconv.ParseUint(port, 10, 16)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			nodeGateway := "http://localhost:" + nodeListenPort.Text

			go listenWallet(uint16(p), nodeGateway, stopWallet)
			toggleWalletRunButton.SetText("Stop wallet ui")
		}
		walletRunning = !walletRunning
	})
	walletUi := container.NewGridWithColumns(2, walletListenPort, toggleWalletRunButton)

	var data = [][]string{[]string{"host", "port", "status"},
		[]string{"0.0.0.0", "30000", "active"},
		[]string{"0.0.0.0", "30001", "deactive"},
		[]string{"0.0.0.0", "30002", "active"},
		[]string{"0.0.0.0", "30003", "active"},
		[]string{"0.0.0.0", "30004", "deactive"},
		[]string{"0.0.0.0", "30005", "active"},
		[]string{"0.0.0.0", "30006", "active"},
		[]string{"0.0.0.0", "30007", "active"},
		[]string{"0.0.0.0", "30008", "active"},
		[]string{"0.0.0.0", "30009", "deactive"},
	}
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
	list.Resize(fyne.NewSize(640, 480))

	logsWidget.Disable()
	logsWidget.TextStyle.Monospace = true

	logsCard := widget.NewCard("Logs", "", logsWidget)
	listCard := widget.NewCard("Nodes", "", list)
	split := container.NewVSplit(listCard, logsCard)
	split.Offset = 0.4
	w.SetContent(split)

	vBox := container.NewVBox(
		nodeLIsten,
		walletUi,
		openBlockExplorer,
		startMining,
	)

	panel := container.NewBorder(vBox, nil, nil, nil, split)
	w.SetContent(panel)
	w.Resize(fyne.NewSize(940, 680))
	w.SetFixedSize(true)
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
