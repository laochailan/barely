package maildir

import "testing"

func TestFlagging(t *testing.T) {
	test := []string{
		"/dir/1439037327_0.709.test,U=2252,FMD5=7e33426f1e6e9d79b29c3f82c57e:2,RS",
		"/dur/1439038239_0.709.test,U=2253,FMD5=7e33426f1e6e9d79b29c3f82c57e:2,",
		"/dur/1439038239_0.709.test,U=2253,FMD5=7e33426f1e6e9d79b29c3f82c57e",
	}

	want := []string{
		"/dir/1439037327_0.709.test,U=2252,FMD5=7e33426f1e6e9d79b29c3f82c57e:2,S",
		"/dir/1439037327_0.709.test,U=2252,FMD5=7e33426f1e6e9d79b29c3f82c57e:2,RS",
		"/dur/1439038239_0.709.test,U=2253,FMD5=7e33426f1e6e9d79b29c3f82c57e:2,",
		"/dur/1439038239_0.709.test,U=2253,FMD5=7e33426f1e6e9d79b29c3f82c57e:2,R",
	}

	r, err := flagName(test[0], 'R', false)
	if r != want[0] {
		t.Fail()
	}
	r, err = flagName(test[0], 'R', true)
	if r != want[1] {
		t.Fail()
	}
	r, err = flagName(test[1], 'R', false)
	if r != want[2] {
		t.Fail()
	}
	r, err = flagName(test[1], 'R', true)
	if r != want[3] {
		t.Fail()
	}
	r, err = flagName(test[2], 'R', true)
	if err == nil {
		t.Fail()
	}
}
