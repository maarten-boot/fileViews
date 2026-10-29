//go:build unix

package listing

import (
	"io/fs"
	"os/user"
	"strconv"
	"sync"
	"syscall"
)

var (
	nameCacheMu sync.Mutex
	userNames   = map[uint32]string{}
	groupNames  = map[uint32]string{}
)

// ownership returns the hard link count, owner and group of info.
// Unknown ids are shown numerically (FR-022).
func ownership(info fs.FileInfo) (links uint64, owner, group string) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, "", ""
	}
	return uint64(st.Nlink), userName(st.Uid), groupName(st.Gid)
}

func userName(uid uint32) string {
	return cachedName(userNames, uid, func(id string) (string, error) {
		u, err := user.LookupId(id)
		if err != nil {
			return "", err
		}
		return u.Username, nil
	})
}

func groupName(gid uint32) string {
	return cachedName(groupNames, gid, func(id string) (string, error) {
		g, err := user.LookupGroupId(id)
		if err != nil {
			return "", err
		}
		return g.Name, nil
	})
}

func cachedName(cache map[uint32]string, id uint32, lookup func(string) (string, error)) string {
	nameCacheMu.Lock()
	defer nameCacheMu.Unlock()
	if name, ok := cache[id]; ok {
		return name
	}
	s := strconv.FormatUint(uint64(id), 10)
	name, err := lookup(s)
	if err != nil {
		name = s
	}
	cache[id] = name
	return name
}

// FreeSpace returns the bytes available to unprivileged users on the filesystem holding path (FR-050).
func FreeSpace(path string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	return st.Bavail * uint64(st.Bsize), nil
}
