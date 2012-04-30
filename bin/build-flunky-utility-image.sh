#!/bin/bash
# add -d option to specify work directory (and selectively download files)

HECKLE_SRC=`dirname $0`

MASTER_URL="http://distro.ibiblio.org/tinycorelinux/4.x/x86/release/distribution_files/"

tmpdir="/tmp/fi.tmp.$$"
echo "tmpdir is $tmpdir"

datestr=`date +"%y%m%d"`

sudo echo "Acquired sudo session"

mkdir $tmpdir
echo "downloading kernel."
wget -nv "${MASTER_URL}/vmlinuz64" -O "$tmpdir/futil-kernel-$datestr"
echo "downloading ramdisk."
wget -nv "${MASTER_URL}/core64.gz" -O "$tmpdir/core64.gz"

cd $tmpdir
mkdir extract 
pushd extract
#cat core64.gz | sudo cpio -i -H newc -d
gzip -dc ../core64.gz | sudo cpio -i -d -H newc
popd

echo "setting up tcroot for local package installs"
sudo cp /etc/resolv.conf extract/etc
sudo mount -o bind /proc extract/proc
sudo chroot extract mkdir -p /etc/sysconfig/tcedir/optional
sudo chroot extract chown -R tc /etc/sysconfig/tcedir

for pkg in bc-1.06.94 parted grub2 rsync tar openssh curl ethtool pci-utils lftp firmware bash ; do 
  echo "Installing pkg $pkg"
  sudo chroot extract su tc -c "/usr/bin/tce-load -w -c -i ${pkg}.tcz"
done

sudo rm extract/etc/resolv.conf
sudo umount extract/proc
sudo chroot extract sh -c "echo \"ttyS0::respawn:/sbin/rungetty ttyS0 --autologin root\" >> /etc/inittab"
sudo chroot extract sh -c "echo ttyS0 >> /etc/securetty"

if [ -x ${HECKLE_SRC}/bin/flunky ] ; then
   cp -f ${HECKLE_SRC}/bin/flunky extract/bin/flunky
   sudo chroot extract rm /opt/bootlocal.sh
   sudo chroot extract ln -sf /bin/flunky /opt/bootlocal.sh
else
   echo "Could not locate flunky binary"
fi

pushd extract 
sudo rm -Rf etc/sysconfig/tcedir/optional/*.tcz
find | sudo cpio -o -H newc | gzip -2 > ../futil-initrd-$datestr.gz
popd

exit 0

#rm -r $tmpdir

