package primitives

/*
#include <stdlib.h>
#include <sys/types.h>
#include <sys/ipc.h>
#include <sys/msg.h>
#include <sys/shm.h>
key_t ftok(const char *pathname, int proj_id);
*/
import "C"

// IpcPerms holds information about the permissions of a SysV IPC object.
type IpcPerms struct {
	OwnerUID   int
	OwnerGID   int
	CreatorUID int
	CreatorGID int
	Mode       uint16
}
