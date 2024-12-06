package main

import (
	"fmt"
	"math/big"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/MariusVanDerWijden/tx-fuzz/flags"
	"github.com/MariusVanDerWijden/tx-fuzz/spammer"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

var (
	// Mutex to handle concurrent access
	serverMutex sync.Mutex
	running     bool
	cancelFunc  func()
)

var airdropCommand = &cli.Command{
	Name:   "airdrop",
	Usage:  "Airdrops to a list of accounts",
	Action: runAirdrop,
	Flags: []cli.Flag{
		flags.SkFlag,
		flags.RpcFlag,
	},
}

var spamCommand = &cli.Command{
	Name:   "spam",
	Usage:  "Send spam transactions",
	Action: runBasicSpam,
	Flags:  flags.SpamFlags,
}

var blobSpamCommand = &cli.Command{
	Name:   "blobs",
	Usage:  "Send blob spam transactions",
	Action: runBlobSpam,
	Flags:  flags.SpamFlags,
}

var createCommand = &cli.Command{
	Name:   "create",
	Usage:  "Create ephemeral accounts",
	Action: runCreate,
	Flags: []cli.Flag{
		flags.CountFlag,
		flags.RpcFlag,
	},
}

var serverCommand = &cli.Command{
	Name:   "server",
	Usage:  "start a server",
	Action: runServer,
	Flags:  flags.ServerFlags,
}

func runServer(c *cli.Context) error {
	mux := http.NewServeMux()
	config, err := spammer.NewConfigFromContext(c)
	if err != nil {
		return err
	}

	mux.HandleFunc("/spam/start", func(w http.ResponseWriter, r *http.Request) {
		serverMutex.Lock()
		defer serverMutex.Unlock()

		if running {
			http.Error(w, "Spam already running", http.StatusConflict)
			return
		}

		cancel := make(chan struct{})
		cancelFunc = func() {
			select {
			case <-cancel:
				// Already closed
			default:
				close(cancel)
			}
		}

		go func() {
			airdropValue := new(big.Int).Mul(big.NewInt(int64((1+config.N)*1000000)), big.NewInt(params.GWei))
			err := spam(config, spammer.SendBasicTransactions, airdropValue, cancel)
			if err != nil {
				fmt.Println("Error running spam:", err)
			}
		}()
		running = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Spam started"))
	})

	mux.HandleFunc("/spam/stop", func(w http.ResponseWriter, r *http.Request) {
		serverMutex.Lock()
		defer serverMutex.Unlock()

		if !running {
			http.Error(w, "No spam running", http.StatusBadRequest)
			return
		}

		if cancelFunc != nil {
			cancelFunc()
			cancelFunc = nil
		}
		running = false
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Spam stopped"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Healthy"))
	})

	serverAddr := "0.0.0.0:" + config.ListenPort
	fmt.Println("Starting server on", serverAddr)
	return http.ListenAndServe(serverAddr, mux)
}

var unstuckCommand = &cli.Command{
	Name:   "unstuck",
	Usage:  "Tries to unstuck an account",
	Action: runUnstuck,
	Flags:  flags.SpamFlags,
}

func initApp() *cli.App {
	app := cli.NewApp()
	app.Name = "tx-fuzz"
	app.Usage = "Fuzzer for sending spam transactions"
	app.Commands = []*cli.Command{
		airdropCommand,
		spamCommand,
		blobSpamCommand,
		createCommand,
		unstuckCommand,
		serverCommand,
	}
	return app
}

var app = initApp()

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runAirdrop(c *cli.Context) error {
	config, err := spammer.NewConfigFromContext(c)
	if err != nil {
		return err
	}
	txPerAccount := config.N
	airdropValue := new(big.Int).Mul(big.NewInt(int64(txPerAccount*100000)), big.NewInt(params.GWei))
	spammer.Airdrop(config, airdropValue)
	return nil
}

func spam(config *spammer.Config, spamFn spammer.Spam, airdropValue *big.Int, cancel <-chan struct{}) error {
	// Unstuck accounts before starting the spam process
	if err := spammer.Unstuck(config); err != nil {
		return fmt.Errorf("failed to unstuck accounts: %w", err)
	}

	for {
		select {
		case <-cancel:
			// Exit the loop when the cancel signal is received
			fmt.Println("Spam process stopped")
			return nil
		default:
			// Perform the spam logic if no cancel signal
			if err := spammer.Airdrop(config, airdropValue); err != nil {
				return fmt.Errorf("failed to airdrop: %w", err)
			}
			if err := spamFn(config, config.GetFaucet(), nil); err != nil {
				return fmt.Errorf("failed to spam transactions: %w", err)
			}
			time.Sleep(time.Duration(config.SlotTime) * time.Second)
		}
	}
}

func runBasicSpam(c *cli.Context) error {
	config, err := spammer.NewConfigFromContext(c)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	airdropValue := new(big.Int).Mul(big.NewInt(int64((1+config.N)*1000000)), big.NewInt(params.GWei))
	cancel := make(chan struct{})
	defer close(cancel)

	return spam(config, spammer.SendBasicTransactions, airdropValue, cancel)
}

func runBlobSpam(c *cli.Context) error {
	config, err := spammer.NewConfigFromContext(c)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	airdropValue := new(big.Int).Mul(big.NewInt(int64((1+config.N)*1000000)), big.NewInt(params.GWei))
	airdropValue.Mul(airdropValue, big.NewInt(100))
	cancel := make(chan struct{})
	defer close(cancel)

	return spam(config, spammer.SendBlobTransactions, airdropValue, cancel)
}

func runCreate(c *cli.Context) error {
	spammer.CreateAddresses(100)
	return nil
}

func runUnstuck(c *cli.Context) error {
	config, err := spammer.NewConfigFromContext(c)
	if err != nil {
		return err
	}
	return spammer.Unstuck(config)
}
