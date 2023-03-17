package primitives

import (
	"math"
	"os"
	"testing"
)

func TestReadAndWrite(t *testing.T) {
	shmSetup(t)
	defer shmTeardown(t)

	s := "this is a test string"
	s2 := "is a test string this"

	_, err := mount.Write([]byte(s))
	if err != nil {
		t.Fatal(err)
	}

	_, err = mount.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	holder := make([]byte, len(s))
	_, err = mount.Read(holder)
	if err != nil {
		t.Error(err)
	}

	if string(holder) != s {
		t.Errorf("mismatched text, got back %v", holder)
	}

	_, err = mount.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mount.Write([]byte(s2))
	if err != nil {
		t.Fatal(err)
	}

	_, err = mount.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	holder2 := make([]byte, len(s2))
	_, err = mount.Read(holder2)
	if err != nil {
		t.Error(err)
	}

	if string(holder2) != s2 {
		t.Errorf("mismatched text, got back %v", holder2)
	}

	if string(holder) == string(holder2) {
		t.Errorf("old text references new text, text: %v", holder2)
	}
}

func TestReadAndWriteAtomicUint64(t *testing.T) {
	shmSetup(t)
	defer shmTeardown(t)

	const (
		target uint64 = math.MaxUint64 - 1
	)

	err := mount.AtomicWriteUint64(target)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mount.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	val, err := mount.AtomicReadUint64()
	if err != nil {
		t.Fatal(err)
	}

	if val != target {
		t.Fatalf("different values recv. expected: %v, got: %v", target, val)
	}
}

func TestSHMReadOnlyError(t *testing.T) {
	shmSetup(t)
	defer shmTeardown(t)

	roat, err := shm.Attach(&SHMAttachFlags{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := roat.Write([]byte("read only")); err == nil {
		t.Fatal("write should be forbidden")
	}
}

func TestSHMStat(t *testing.T) {
	shmSetup(t)
	defer shmTeardown(t)

	info, err := shm.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Perms.Mode&0777 != 0600 {
		t.Error("wrong permissions", info.Perms.Mode)
	}
	if info.Perms.OwnerUID != os.Getuid() {
		t.Error("wrong owner")
	}
	if info.Perms.CreatorUID != os.Getuid() {
		t.Error("wrong creator")
	}
	if info.SegmentSize != 4096 {
		t.Error("wrong size:", info.SegmentSize)
	}
	if info.CreatorPID != os.Getpid() {
		t.Error("wrong creator pid")
	}
	if info.LastUserPID != os.Getpid() {
		t.Error("wrong last user pid")
	}
	if info.CurrentAttaches != 1 {
		t.Error("wrong number of attaches:", info.CurrentAttaches)
	}

	mnt2, err := shm.Attach(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer mnt2.Close()

	info, err = shm.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.CurrentAttaches != 2 {
		t.Error("missing attach?", info.CurrentAttaches)
	}

	mnt3, err := shm.Attach(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer mnt3.Close()

	info, err = shm.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.CurrentAttaches != 3 {
		t.Error("missing attach?", info.CurrentAttaches)
	}
}

var (
	shm   *SharedMem
	mount *SharedMemMount
)

func shmSetup(t *testing.T) {
	mem, err := GetSharedMem(0xE4CA, 4096, &SHMFlags{
		Create:    true,
		Exclusive: true,
		Perms:     0600,
	})
	if err != nil {
		t.Fatal(err)
	}
	shm = mem

	mnt, err := shm.Attach(nil)
	if err != nil {
		t.Fatal(err)
	}
	mount = mnt

	err = shm.Remove()
	if err != nil {
		t.Fatal(err)
	}
}

func shmTeardown(t *testing.T) {
	mount.Close()
}
