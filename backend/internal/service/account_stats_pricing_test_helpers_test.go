package service

import (
	"testing"
	"time"
)

// newTestChannelServiceForStats returns a channel service backed by one cached channel.
func newTestChannelServiceForStats(t *testing.T, channel *Channel, groupID int64, platform string) *ChannelService {
	t.Helper()
	cache := newEmptyChannelCache()
	cache.channelByGroupID[groupID] = channel
	cache.groupPlatform[groupID] = platform
	cache.loadedAt = time.Now()
	cs := &ChannelService{}
	cs.cache.Store(cache)
	return cs
}
