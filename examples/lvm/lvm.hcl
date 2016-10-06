lvm.vg "mantl" {
  name = "mantl"
  devices = [ "/dev/sdb" ]
}

lvm.lv "lv-test" {
  group = "mantl"
  name = "test"
  size = "1G"
  depends  = [ "lvm.vg.mantl" ]
}

lvm.fs "mnt-me"  {
  device = "/dev/mapper/mantl-test"
  mount = "/mnt"
  type = "xfs"
  depends = [ "lvm.lv.lv-test" ]
}
