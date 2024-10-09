package utils

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"

	lcli "github.com/filecoin-project/lotus/cli"

	powerlib "github.com/filecoin-project/go-state-types/builtin/v9/power"
	"github.com/filecoin-project/go-state-types/builtin/v9/reward"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/urfave/cli/v2"
)

// MinerState
type MinerFullData struct {
	Address           address.Address
	StateHeight       abi.ChainEpoch
	MinerBalance      MinerBalance
	MinerPower        *api.MinerPower
	MinerSectors      MinerSectors
	MinerSectorsState MinerSectorsState
	MinerInfo         miner.MinerInfo
}

type MinerBalance struct {
	Balance           abi.TokenAmount
	AvailableBalance  abi.TokenAmount
	InitialPledge     abi.TokenAmount
	LockedRewards     abi.TokenAmount
	PreCommitDeposits abi.TokenAmount
}

type MinerSectors struct {
	api.MinerSectors
	Recoveries uint64
}

type MinerSectorsState struct {
	CCCount                uint64
	DCCount                uint64
	AllInitialPledge       abi.TokenAmount
	TerminateALLFineReward abi.TokenAmount
	TerminateCCFineReward  abi.TokenAmount
	TerminateDCFineReward  abi.TokenAmount
}

var MinerExCmd = &cli.Command{
	Name:  "miner",
	Usage: "Miner with filecoin blockchain",
	Subcommands: []*cli.Command{
		MinerListCmd,
		MinerStateCmd,
		MinerSectorCmd,
		MinerEstimateFaultySectorCmd,
		CollectMinersSectorCmd,
		CollectMinerSectorCmd,
	},
}

// MinerListCmd  矿工列表
var MinerListCmd = &cli.Command{
	Name:      "list",
	Usage:     "Miner list",
	ArgsUsage: "[miner address]",
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
		if err != nil {
			return err
		}

		out, err := json.MarshalIndent(miners, "", "  ")
		if err != nil {
			return err
		}

		afmt := NewAppFmt(cctx.App)
		afmt.Println(string(out))

		return nil
	},
}

var MinerEstimateFaultySectorCmd = &cli.Command{
	Name:      "estimate-faulty",
	Aliases:   []string{"estimatefaulty"},
	Usage:     "Miner estimatefaulty",
	ArgsUsage: "[miner address]",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:    "pos",
			Aliases: []string{"p"},
			Usage:   "Terminate start pos sectors",
		},
		&cli.IntFlag{
			Name:    "number",
			Aliases: []string{"n"},
			Usage:   "Terminate number sectors",
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify miner to show power for")
		}
		ts, err := lcli.LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		pos := cctx.Int("pos")
		size := cctx.Int("number")

		maddr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}
		mact, err := api.StateGetActor(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}
		tbs := blockstore.NewTieredBstore(blockstore.NewAPIBlockstore(api), blockstore.NewMemory())

		mas, err := miner.Load(adt.WrapStore(ctx, cbor.NewCborStore(tbs)), mact)
		if err != nil {
			return err
		}
		FaultyType, err := miner.AllPartSectors(mas, miner.Partition.FaultySectors)
		if err != nil {
			return err
		}
		faultysectors, err := api.StateMinerSectors(ctx, maddr, &FaultyType, ts.Key())
		if err != nil {
			return err
		}
		// 获取全网奖励
		act, err := api.StateGetActor(ctx, builtin.RewardActorAddr, ts.Key())
		if err != nil {
			return err
		}
		actorHead, err := api.ChainReadObj(ctx, act.Head)
		if err != nil {
			return err
		}
		var rewardActorState reward.State
		if err := rewardActorState.UnmarshalCBOR(bytes.NewReader(actorHead)); err != nil {
			return err
		}
		// 获取全网算力
		actst, err := api.StateGetActor(ctx, builtin.StoragePowerActorAddr, ts.Key())
		if err != nil {
			return err
		}
		stactorHead, err := api.ChainReadObj(ctx, actst.Head)
		if err != nil {
			return err
		}
		var powerActorState powerlib.State
		if err := powerActorState.UnmarshalCBOR(bytes.NewReader(stactorHead)); err != nil {
			return err
		}

		estimatefaulty := struct {
			CurHeight        abi.ChainEpoch
			Address          address.Address
			TotalFaultyCount int
			Terminate        struct {
				Count      int
				LostPower  string
				FineReward string
				Sectors    []*miner.SectorOnChainInfo
			}
		}{}

		minerInfo, err := mas.Info()
		if err != nil {
			return err
		}
		estimatefaulty.CurHeight = ts.Height()
		estimatefaulty.Address = maddr
		estimatefaulty.TotalFaultyCount = len(faultysectors)
		estimatefaulty.Terminate.Count = size
		if pos >= 0 && (pos+size) <= len(faultysectors) {
			var tfaultysectors []*miner.SectorOnChainInfo
			for i, faultysector := range faultysectors {
				if i < pos {
					continue
				}
				if i >= pos+size {
					break
				}
				tfaultysectors = append(tfaultysectors, faultysector)
			}
			faultysectors = tfaultysectors
			estimatefaulty.Terminate.Count = len(faultysectors)
		}
		estimatefaulty.Terminate.Sectors = faultysectors

		estimatefaulty.Terminate.LostPower = types.SizeStr(big.Mul(big.NewInt(int64(len(faultysectors))), big.NewIntUnsigned(uint64(minerInfo.SectorSize))))
		estimatefaulty.Terminate.FineReward = types.FIL(terminationPenalty(ts.Height(), rewardActorState.ThisEpochRewardSmoothed, powerActorState.ThisEpochQAPowerSmoothed, faultysectors)).Short()

		out, err := json.MarshalIndent(estimatefaulty, "", "  ")
		if err != nil {
			return err
		}
		afmt := NewAppFmt(cctx.App)
		afmt.Println(string(out))

		return nil
	},
}

var MinerStateCmd = &cli.Command{
	Name:      "state",
	Usage:     "Miner state",
	ArgsUsage: "[miner address]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "calc-terminate",
			Aliases: []string{"s"},
			Usage:   "calc MinerSectorsState",
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify miner to show power for")
		}

		ts, err := lcli.LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		var minerFullData MinerFullData
		minerFullData.StateHeight = ts.Height()

		maddr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		minerFullData.Address = maddr

		walletBalance, err := api.WalletBalance(ctx, maddr)
		if err != nil {
			return err
		}
		minerFullData.MinerBalance.Balance = walletBalance

		availableBalance, err := api.StateMinerAvailableBalance(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}
		minerFullData.MinerBalance.AvailableBalance = availableBalance

		mact, err := api.StateGetActor(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}

		tbs := blockstore.NewTieredBstore(blockstore.NewAPIBlockstore(api), blockstore.NewMemory())

		mas, err := miner.Load(adt.WrapStore(ctx, cbor.NewCborStore(tbs)), mact)
		if err != nil {
			return err
		}

		LockedFunds, _ := mas.LockedFunds()

		minerFullData.MinerBalance.InitialPledge = LockedFunds.InitialPledgeRequirement
		minerFullData.MinerBalance.LockedRewards = LockedFunds.VestingFunds
		minerFullData.MinerBalance.PreCommitDeposits = LockedFunds.PreCommitDeposits

		power, err := api.StateMinerPower(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}
		minerFullData.MinerPower = power

		minerSectors, err := api.StateMinerSectorCount(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}
		recoveries, err := api.StateMinerRecoveries(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}
		minerFullData.MinerSectors.MinerSectors = minerSectors
		minerFullData.MinerSectors.Recoveries, _ = recoveries.Count()

		if !cctx.Bool("s") {
			// 获取全网奖励
			act, err := api.StateGetActor(ctx, builtin.RewardActorAddr, ts.Key())
			if err != nil {
				return err
			}
			actorHead, err := api.ChainReadObj(ctx, act.Head)
			if err != nil {
				return err
			}
			var rewardActorState reward.State
			if err := rewardActorState.UnmarshalCBOR(bytes.NewReader(actorHead)); err != nil {
				return err
			}

			// 获取全网算力
			actst, err := api.StateGetActor(ctx, builtin.StoragePowerActorAddr, ts.Key())
			if err != nil {
				return err
			}
			stactorHead, err := api.ChainReadObj(ctx, actst.Head)
			if err != nil {
				return err
			}
			var powerActorState powerlib.State
			if err := powerActorState.UnmarshalCBOR(bytes.NewReader(stactorHead)); err != nil {
				return err
			}

			var dcsectors, ccsectors []*miner.SectorOnChainInfo
			liveType, err := miner.AllPartSectors(mas, miner.Partition.LiveSectors)
			if err != nil {
				return err
			}
			liveSectors, err := api.StateMinerSectors(ctx, maddr, &liveType, ts.Key())
			if err != nil {
				return err
			}

			minerFullData.MinerSectorsState.AllInitialPledge = big.Zero()
			for _, s := range liveSectors {
				minerFullData.MinerSectorsState.AllInitialPledge = big.Add(minerFullData.MinerSectorsState.AllInitialPledge, s.InitialPledge)
				if len(s.DealIDs) > 0 {
					minerFullData.MinerSectorsState.DCCount++
					dcsectors = append(dcsectors, s)
				} else {
					minerFullData.MinerSectorsState.CCCount++
					ccsectors = append(ccsectors, s)
				}
			}

			minerFullData.MinerSectorsState.TerminateALLFineReward = terminationPenalty(ts.Height(), rewardActorState.ThisEpochRewardSmoothed, powerActorState.ThisEpochQAPowerSmoothed, liveSectors)
			if len(dcsectors) > len(ccsectors) {
				if len(dcsectors) == len(liveSectors) {
					minerFullData.MinerSectorsState.TerminateDCFineReward = minerFullData.MinerSectorsState.TerminateALLFineReward
					minerFullData.MinerSectorsState.TerminateCCFineReward = big.Zero()
				} else {
					minerFullData.MinerSectorsState.TerminateCCFineReward = terminationPenalty(ts.Height(), rewardActorState.ThisEpochRewardSmoothed, powerActorState.ThisEpochQAPowerSmoothed, ccsectors)
					minerFullData.MinerSectorsState.TerminateDCFineReward = big.Sub(minerFullData.MinerSectorsState.TerminateALLFineReward, minerFullData.MinerSectorsState.TerminateCCFineReward)
				}
			} else {
				if len(ccsectors) == len(liveSectors) {
					minerFullData.MinerSectorsState.TerminateCCFineReward = minerFullData.MinerSectorsState.TerminateALLFineReward
					minerFullData.MinerSectorsState.TerminateDCFineReward = big.Zero()
				} else {
					minerFullData.MinerSectorsState.TerminateDCFineReward = terminationPenalty(ts.Height(), rewardActorState.ThisEpochRewardSmoothed, powerActorState.ThisEpochQAPowerSmoothed, dcsectors)
					minerFullData.MinerSectorsState.TerminateCCFineReward = big.Sub(minerFullData.MinerSectorsState.TerminateALLFineReward, minerFullData.MinerSectorsState.TerminateDCFineReward)
				}
			}
		}

		minerInfo, err := mas.Info()
		if err != nil {
			return err
		}
		minerFullData.MinerInfo = minerInfo

		out, err := json.MarshalIndent(minerFullData, "", "  ")
		if err != nil {
			return err
		}

		afmt := NewAppFmt(cctx.App)
		afmt.Println(string(out))

		return nil
	},
}