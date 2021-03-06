package bank

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

const (
	costGetCoins      sdk.Gas = 10
	costHasCoins      sdk.Gas = 10
	costSetCoins      sdk.Gas = 100
	costSubtractCoins sdk.Gas = 10
	costAddCoins      sdk.Gas = 10
)

// Keeper manages transfers between accounts
type Keeper struct {
	am auth.AccountMapper
}

// NewKeeper returns a new Keeper
func NewKeeper(am auth.AccountMapper) Keeper {
	return Keeper{am: am}
}

// GetCoins returns the coins at the addr.
func (keeper Keeper) GetCoins(ctx sdk.Context, addr sdk.Address) sdk.Coins {
	return getCoins(ctx, keeper.am, addr)
}

// SetCoins sets the coins at the addr.
func (keeper Keeper) SetCoins(ctx sdk.Context, addr sdk.Address, amt sdk.Coins) sdk.Error {
	return setCoins(ctx, keeper.am, addr, amt)
}

// HasCoins returns whether or not an account has at least amt coins.
func (keeper Keeper) HasCoins(ctx sdk.Context, addr sdk.Address, amt sdk.Coins) bool {
	return hasCoins(ctx, keeper.am, addr, amt)
}

// SubtractCoins subtracts amt from the coins at the addr.
func (keeper Keeper) SubtractCoins(ctx sdk.Context, addr sdk.Address, amt sdk.Coins) (sdk.Coins, sdk.Tags, sdk.Error) {
	return subtractCoins(ctx, keeper.am, addr, amt)
}

// AddCoins adds amt to the coins at the addr.
func (keeper Keeper) AddCoins(ctx sdk.Context, addr sdk.Address, amt sdk.Coins) (sdk.Coins, sdk.Tags, sdk.Error) {
	return addCoins(ctx, keeper.am, addr, amt)
}

// SendCoins moves coins from one account to another
func (keeper Keeper) SendCoins(ctx sdk.Context, fromAddr sdk.Address, toAddr sdk.Address, amt sdk.Coins) (sdk.Tags, sdk.Error) {
	return sendCoins(ctx, keeper.am, fromAddr, toAddr, amt)
}

// InputOutputCoins handles a list of inputs and outputs
func (keeper Keeper) InputOutputCoins(ctx sdk.Context, inputs []Input, outputs []Output) (sdk.Tags, sdk.Error) {
	return inputOutputCoins(ctx, keeper.am, inputs, outputs)
}

//______________________________________________________________________________________________

// SendKeeper only allows transfers between accounts, without the possibility of creating coins
type SendKeeper struct {
	am auth.AccountMapper
}

// NewSendKeeper returns a new Keeper
func NewSendKeeper(am auth.AccountMapper) SendKeeper {
	return SendKeeper{am: am}
}

// GetCoins returns the coins at the addr.
func (keeper SendKeeper) GetCoins(ctx sdk.Context, addr sdk.Address) sdk.Coins {
	return getCoins(ctx, keeper.am, addr)
}

// HasCoins returns whether or not an account has at least amt coins.
func (keeper SendKeeper) HasCoins(ctx sdk.Context, addr sdk.Address, amt sdk.Coins) bool {
	return hasCoins(ctx, keeper.am, addr, amt)
}

// SendCoins moves coins from one account to another
func (keeper SendKeeper) SendCoins(ctx sdk.Context, fromAddr sdk.Address, toAddr sdk.Address, amt sdk.Coins) (sdk.Tags, sdk.Error) {
	return sendCoins(ctx, keeper.am, fromAddr, toAddr, amt)
}

// InputOutputCoins handles a list of inputs and outputs
func (keeper SendKeeper) InputOutputCoins(ctx sdk.Context, inputs []Input, outputs []Output) (sdk.Tags, sdk.Error) {
	return inputOutputCoins(ctx, keeper.am, inputs, outputs)
}

//______________________________________________________________________________________________

// ViewKeeper only allows reading of balances
type ViewKeeper struct {
	am auth.AccountMapper
}

// NewViewKeeper returns a new Keeper
func NewViewKeeper(am auth.AccountMapper) ViewKeeper {
	return ViewKeeper{am: am}
}

// GetCoins returns the coins at the addr.
func (keeper ViewKeeper) GetCoins(ctx sdk.Context, addr sdk.Address) sdk.Coins {
	return getCoins(ctx, keeper.am, addr)
}

// HasCoins returns whether or not an account has at least amt coins.
func (keeper ViewKeeper) HasCoins(ctx sdk.Context, addr sdk.Address, amt sdk.Coins) bool {
	return hasCoins(ctx, keeper.am, addr, amt)
}

//______________________________________________________________________________________________

func getCoins(ctx sdk.Context, am auth.AccountMapper, addr sdk.Address) sdk.Coins {
	ctx.GasMeter().ConsumeGas(costGetCoins, "getCoins")
	acc := am.GetAccount(ctx, addr)
	if acc == nil {
		return sdk.Coins{}
	}
	return acc.GetCoins()
}

func setCoins(ctx sdk.Context, am auth.AccountMapper, addr sdk.Address, amt sdk.Coins) sdk.Error {
	ctx.GasMeter().ConsumeGas(costSetCoins, "setCoins")
	acc := am.GetAccount(ctx, addr)
	if acc == nil {
		acc = am.NewAccountWithAddress(ctx, addr)
	}
	acc.SetCoins(amt)
	am.SetAccount(ctx, acc)
	return nil
}

// HasCoins returns whether or not an account has at least amt coins.
func hasCoins(ctx sdk.Context, am auth.AccountMapper, addr sdk.Address, amt sdk.Coins) bool {
	ctx.GasMeter().ConsumeGas(costHasCoins, "hasCoins")
	return getCoins(ctx, am, addr).IsGTE(amt)
}

// SubtractCoins subtracts amt from the coins at the addr.
func subtractCoins(ctx sdk.Context, am auth.AccountMapper, addr sdk.Address, amt sdk.Coins) (sdk.Coins, sdk.Tags, sdk.Error) {
	ctx.GasMeter().ConsumeGas(costSubtractCoins, "subtractCoins")
	oldCoins := getCoins(ctx, am, addr)
	newCoins := oldCoins.Minus(amt)
	if !newCoins.IsNotNegative() {
		return amt, nil, sdk.ErrInsufficientCoins(fmt.Sprintf("%s < %s", oldCoins, amt))
	}
	err := setCoins(ctx, am, addr, newCoins)
	tags := sdk.NewTags("sender", []byte(addr.String()))
	return newCoins, tags, err
}

// AddCoins adds amt to the coins at the addr.
func addCoins(ctx sdk.Context, am auth.AccountMapper, addr sdk.Address, amt sdk.Coins) (sdk.Coins, sdk.Tags, sdk.Error) {
	ctx.GasMeter().ConsumeGas(costAddCoins, "addCoins")
	oldCoins := getCoins(ctx, am, addr)
	newCoins := oldCoins.Plus(amt)
	if !newCoins.IsNotNegative() {
		return amt, nil, sdk.ErrInsufficientCoins(fmt.Sprintf("%s < %s", oldCoins, amt))
	}
	err := setCoins(ctx, am, addr, newCoins)
	tags := sdk.NewTags("recipient", []byte(addr.String()))
	return newCoins, tags, err
}

// SendCoins moves coins from one account to another
// NOTE: Make sure to revert state changes from tx on error
func sendCoins(ctx sdk.Context, am auth.AccountMapper, fromAddr sdk.Address, toAddr sdk.Address, amt sdk.Coins) (sdk.Tags, sdk.Error) {
	_, subTags, err := subtractCoins(ctx, am, fromAddr, amt)
	if err != nil {
		return nil, err
	}

	_, addTags, err := addCoins(ctx, am, toAddr, amt)
	if err != nil {
		return nil, err
	}

	return subTags.AppendTags(addTags), nil
}

// InputOutputCoins handles a list of inputs and outputs
// NOTE: Make sure to revert state changes from tx on error
func inputOutputCoins(ctx sdk.Context, am auth.AccountMapper, inputs []Input, outputs []Output) (sdk.Tags, sdk.Error) {
	allTags := sdk.EmptyTags()

	for _, in := range inputs {
		_, tags, err := subtractCoins(ctx, am, in.Address, in.Coins)
		if err != nil {
			return nil, err
		}
		allTags = allTags.AppendTags(tags)
	}

	for _, out := range outputs {
		_, tags, err := addCoins(ctx, am, out.Address, out.Coins)
		if err != nil {
			return nil, err
		}
		allTags = allTags.AppendTags(tags)
	}

	return allTags, nil
}

//*****comdex
func getAssetWallet(ctx sdk.Context, am auth.AccountMapper, addr sdk.Address) sdk.AssetPegWallet {
	acc := am.GetAccount(ctx, addr)
	if acc == nil {
		return sdk.AssetPegWallet{}
	}
	return acc.GetAssetPegWallet()
}

func setAssetWallet(ctx sdk.Context, am auth.AccountMapper, addr sdk.Address, asset sdk.AssetPegWallet) sdk.Error {
	acc := am.GetAccount(ctx, addr)
	if acc == nil {
		acc = am.NewAccountWithAddress(ctx, addr)
	}
	acc.SetAssetPegWallet(asset)
	am.SetAccount(ctx, acc)
	return nil
}

func instantiateAndAssignAsset(ctx sdk.Context, am auth.AccountMapper, issuerAddress sdk.Address, toAddress sdk.Address, assetPegWallet sdk.AssetPegWallet) (sdk.AssetPegWallet, sdk.Tags, sdk.Error) {
	issuerOldAssetWallet := getAssetWallet(ctx, am, issuerAddress)
	if len(issuerOldAssetWallet) == 0 {
		return issuerOldAssetWallet, nil, sdk.ErrInsufficientCoins(fmt.Sprintf("no assets left"))
	}
	issuedAsset := issuerOldAssetWallet[len(issuerOldAssetWallet)-1]
	issuerNewAssetWallet := issuerOldAssetWallet[:len(issuerOldAssetWallet)-1]
	toOldAssetWallet := getAssetWallet(ctx, am, toAddress)
	assetPegWallet[0].PegHash = issuedAsset.PegHash
	issuedAsset = assetPegWallet[0]
	toNewAssetWallet := append(toOldAssetWallet, issuedAsset)
	err := setAssetWallet(ctx, am, issuerAddress, issuerNewAssetWallet)
	if err == nil {
		err = setAssetWallet(ctx, am, toAddress, toNewAssetWallet)
		if err != nil {
			setAssetWallet(ctx, am, issuerAddress, issuerOldAssetWallet)
		}
	}
	tags := sdk.NewTags("recepient", []byte(toAddress.String()))
	tags.AppendTag("issuer", []byte(issuerAddress.String()))
	return sdk.AssetPegWallet{issuedAsset}, tags, err
}

func issueAssetsToWallets(ctx sdk.Context, am auth.AccountMapper, issueAsset []IssueAsset) (sdk.Tags, sdk.Error) {
	allTags := sdk.EmptyTags()

	for _, req := range issueAsset {
		_, tags, err := instantiateAndAssignAsset(ctx, am, req.IssuerAddress, req.ToAddress, req.AssetPegWallet)
		if err != nil {
			return nil, err
		}
		allTags = allTags.AppendTags(tags)
	}
	return allTags, nil
}

//IssueAssetsToWallets haddles a list of IssueAsset messages
func (keeper Keeper) IssueAssetsToWallets(ctx sdk.Context, issueAssets []IssueAsset) (sdk.Tags, sdk.Error) {
	return issueAssetsToWallets(ctx, keeper.am, issueAssets)
}

//#####comdex
