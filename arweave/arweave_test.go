package arweave

import (
	"context"
	"fmt"
	"github.com/test-go/testify/require"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"testing"
)

func Test_ArweaveUpload(t *testing.T) {
	walletpath := os.Getenv("ARWEAVE_WALLET")
	if walletpath == "" {
		t.Skipf("set env 'ARWEAVE_WALLET' to run the test")
	}
	testImage := "./testdata/metadata.json"
	ctx := context.Background()

	zlog, _ := zap.NewDevelopment()
	arweave, err := NewArweave("arweave.net", WithLogger(zlog), WithWalletFilepath(walletpath))
	require.NoError(t, err)

	cnt, err := ioutil.ReadFile(testImage)
	require.NoError(t, err)
	tx, err := arweave.Upload(ctx, cnt)
	require.NoError(t, err)
	fmt.Println(tx.ID())
}

func Test_ArweaveUploadAndConfirm(t *testing.T) {
	//walletpath := os.Getenv("ARWEAVE_WALLET")
	//if walletpath == "" {
	//	t.Skipf("set env 'ARWEAVE_WALLET' to run the test")
	//}
	//testImage := "./testdata/metadata.json"
	//ctx := context.Background()
	//
	//zlog, _ := zap.NewDevelopment()
	//arweave, err := NewArweave("arweave.net", WithLogger(zlog), WithWalletFilepath(walletpath))
	//require.NoError(t, err)
	//
	//cnt, err := ioutil.ReadFile(testImage)
	//require.NoError(t, err)
	//tx, err := arweave.UploadAndConfirm(ctx, cnt)
	//require.NoError(t, err)
	//spew.Dump(tx)
	//fmt.Println(tx.ID())
}

//
//func Test_ClientLastTransaction(t *testing.T) {
//	client := NewClient(fmt.Sprintf("%s://%s:%d", "https", "arweave.net", 443))
//	id, err := client.LastTransaction(context.Background(), "PDnnZGG8XlUK7sSVL6P0I6mXdLuQB-wq48UsW4c_WDA")
//	require.NoError(t, err)
//	fmt.Println(id.String())
//
//	tx, err := client.GetTransaction(context.Background(), "2K2iQNDtm2sJLEzQ-Ub_esUIzLT0vdfoSBZ0LUxV-kk")
//	require.NoError(t, err)
//	spew.Dump(tx)
//}
