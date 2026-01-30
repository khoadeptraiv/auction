package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Scripts struct {
	rdb      *redis.Client
	placeBid *redis.Script
}

func NewScripts(rdb *redis.Client) *Scripts {
	return &Scripts{
		rdb:      rdb,
		placeBid: redis.NewScript(placeBidScript),
	}
}

// PlaceBidScript atomically checks and places a bid
// KEYS[1] = auction:{id}:price
// KEYS[2] = auction:{id}:bidder
// ARGV[1] = new bid amount
// ARGV[2] = min increment
// ARGV[3] = bidder ID
// Returns: 1 = success, 0 = bid too low, -1 = auction not found
const placeBidScript = `
local currentPrice = redis.call("GET", KEYS[1])
if not currentPrice then
    return -1
end

local newBid = tonumber(ARGV[1])
local minIncrement = tonumber(ARGV[2])
local currentPriceNum = tonumber(currentPrice)

if newBid >= currentPriceNum + minIncrement then
    redis.call("SET", KEYS[1], newBid)
    redis.call("SET", KEYS[2], ARGV[3])
    return 1
end
return 0
`

func (s *Scripts) PlaceBid(ctx context.Context, auctionID string, amount float64, minIncrement float64, bidderID string) (int64, error) {
	priceKey := "auction:" + auctionID + ":price"
	bidderKey := "auction:" + auctionID + ":bidder"

	result, err := s.placeBid.Run(ctx, s.rdb, []string{priceKey, bidderKey}, amount, minIncrement, bidderID).Int64()
	if err != nil {
		return 0, err
	}
	return result, nil
}

func (s *Scripts) InitAuction(ctx context.Context, auctionID string, startingPrice float64) error {
	priceKey := "auction:" + auctionID + ":price"
	return s.rdb.Set(ctx, priceKey, startingPrice, 0).Err()
}

func (s *Scripts) GetCurrentBid(ctx context.Context, auctionID string) (float64, string, error) {
	priceKey := "auction:" + auctionID + ":price"
	bidderKey := "auction:" + auctionID + ":bidder"

	price, err := s.rdb.Get(ctx, priceKey).Float64()
	if err == redis.Nil {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", err
	}

	bidder, _ := s.rdb.Get(ctx, bidderKey).Result()
	return price, bidder, nil
}
