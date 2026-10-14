package datadir

// beforeEntryCheck, if set, runs before an entry of the data directory is
// inspected (tests remove the entry there).
var beforeEntryCheck func(path string)
