package wallet

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type WalletData struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type Wallet struct {
	Client  *rpc.Client
	Account *solana.Wallet
	DataDir string
}

func NewWallet(dataDir string) (*Wallet, error) {
	w := &Wallet{
		Client:  rpc.New(rpc.DevNet_RPC),
		DataDir: dataDir,
	}

	err := w.loadOrGenerate()
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *Wallet) loadOrGenerate() error {
	walletFile := w.DataDir + "/wallet.json"

	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		log.Println("No wallet found, generating new one...")
		account := solana.NewWallet()
		w.Account = account

		wd := WalletData{
			PrivateKey: account.PrivateKey.String(),
			PublicKey:  account.PublicKey().String(),
		}

		bytes, err := json.MarshalIndent(wd, "", "  ")
		if err != nil {
			return err
		}

		err = os.WriteFile(walletFile, bytes, 0600)
		if err != nil {
			return err
		}
		log.Println("Wallet generated and saved.")
		return nil
	}

	log.Println("Loading existing wallet...")
	bytes, err := os.ReadFile(walletFile)
	if err != nil {
		return err
	}

	var wd WalletData
	err = json.Unmarshal(bytes, &wd)
	if err != nil {
		return err
	}

	privKey, err := solana.PrivateKeyFromBase58(wd.PrivateKey)
	if err != nil {
		return err
	}

	w.Account, err = solana.WalletFromPrivateKeyBase58(privKey.String())
	if err != nil {
	    return err
	}

	log.Printf("Wallet loaded. Public Key: %s", w.Account.PublicKey().String())
	return nil
}

func (w *Wallet) GetBalance() (float64, error) {
	balanceResult, err := w.Client.GetBalance(
		context.TODO(),
		w.Account.PublicKey(),
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return 0, err
	}

	return float64(balanceResult.Value) / 1e9, nil
}

func (w *Wallet) RequestAirdrop() (string, error) {
	log.Println("Requesting airdrop...")
	out, err := w.Client.RequestAirdrop(
		context.TODO(),
		w.Account.PublicKey(),
		1e9, // 1 SOL
		rpc.CommitmentFinalized,
	)

	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func (w *Wallet) GetPublicKey() string {
	return w.Account.PublicKey().String()
}
