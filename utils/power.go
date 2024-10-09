package utils

import (
	"encoding/json"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/urfave/cli/v2"

	lcli "github.com/filecoin-project/lotus/cli"
)

type EXTotalPowerstruct struct {
	Power    abi.StoragePower
	PowerStr string
}

var PowerExCmd = &cli.Command{
	Name:      "power",
	Usage:     "Get TotalPower",
	ArgsUsage: "address",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		ts, err := lcli.LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		miners, err := api.StateListMiners(ctx, ts.Key())
		if err != nil || len(miners) <= 0 {
			return err
		}

		power, err := api.StateMinerPower(ctx, miners[0], ts.Key())
		if err != nil {
			return err
		}

		tp := power.TotalPower

		out := EXTotalPowerstruct{
			Power:    tp.QualityAdjPower,
			PowerStr: types.SizeStr(tp.QualityAdjPower),
		}

		byte, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		afmt := NewAppFmt(cctx.App)
		afmt.Println(string(byte))
		return nil
	},
}
