package helpers

import (
	"strconv"
	"sync"
	"testing"
)

func TestNameResolverCacheConcurrentAccess(t *testing.T) {
	t.Parallel()

	nr := &NameResolver{
		cache:           make(map[string]map[string]map[string]string),
		cloudfrontCache: make(map[string]string),
	}

	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines * 4)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			nr.storeCached("us-east-1", "kms", map[string]string{"id": strconv.Itoa(i)})
		}()
		go func() {
			defer wg.Done()
			_, _ = nr.loadCached("us-east-1", "kms")
		}()
		go func() {
			defer wg.Done()
			nr.storeCloudFrontCached("cachepolicy:"+strconv.Itoa(i), "name")
		}()
		go func() {
			defer wg.Done()
			_, _ = nr.loadCloudFrontCached("cachepolicy:" + strconv.Itoa(i))
		}()
	}

	wg.Wait()
}
