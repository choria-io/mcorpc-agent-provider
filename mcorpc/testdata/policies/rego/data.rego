package io.choria.mcorpc.authpolicy

default allow = false

allow {
	# input.callerID = "choria=ginkgo.mcollective"
    # only allow if the "foo" paramter is equal to bar
    # This is highly context 
    input.data.foo = "bar"
}
