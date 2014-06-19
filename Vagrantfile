# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_API_VERSION = '2'

$script = <<SCRIPT
if [[ ! -e /etc/ssh/.ssh_regen_done ]]; then
  echo "#{File.read "#{Dir.home}/.ssh/id_rsa.pub"}" > /home/vagrant/.ssh/authorized_keys
  chown -R vagrant:vagrant /home/vagrant/.ssh
  sed -ri 's|#PasswordAuthentication yes|PasswordAuthentication no|' /etc/ssh/sshd_config
  rm -rf /etc/ssh/ssh_host_*
  ssh-keygen -A
  mkdir /home/vagrant/.ssh
  chmod 0700 /home/vagrant/.ssh
  ssh-keygen -b 2048 -t rsa -C 'vagrant@ferry' -f /home/vagrant/.ssh/id_rsa -N ''
  chown -R vagrant:vagrant /home/vagrant/.ssh
  service ssh restart
  touch /etc/ssh/.ssh_regen_done
fi

export VAGRANT_HOME=/home/vagrant
export GOPATH=$VAGRANT_HOME/go
export GOROOT=/usr/local/opt/go

apt-get install -yy --no-install-recommends build-essential curl git-core mercurial

echo "Installing Docker 1.0..."
curl -s https://get.docker.io/ubuntu/ | sudo sh

install_go() {
  echo "Installing Go..."
  mkdir -p $GOROOT
  curl -s https://storage.googleapis.com/golang/go1.2.2.linux-amd64.tar.gz | tar -v -C /usr/local/opt -xz
}

if [[ ! -e $GOROOT/bin/go ]]; then
  install_go
  echo "export GOPATH=$GOPATH" >> /etc/profile
  echo "export GOROOT=$GOROOT" >> /etc/profile
  echo "export PATH=$GOPATH/bin:$GOROOT/bin:/usr/local/opt/go/bin:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin" >> /etc/profile
else
  if [[ ! $(go version | grep go1.2) ]]; then
    install_go
    rm -rf $GOPATH/pkg $GOPATH/bin
  fi
fi

if [ ! -h $VAGRANT_HOME/ferry ]; then
  ln -fs $VAGRANT_HOME/go/src/github.com/nitrous-io/ferry $VAGRANT_HOME/ferry
fi

cat > /etc/default/docker.io <<EOF
#Docker Upstart and SysVinit configuration file

# Customize location of Docker binary (especially for development testing).
#DOCKER="/usr/local/bin/docker"

# Use DOCKER_OPTS to modify the daemon startup options.
DOCKER_OPTS="-dns 8.8.8.8 -dns 8.8.4.4 -e lxc"

# If you need Docker to use an HTTP proxy, it can also be specified here.
#export http_proxy="http://127.0.0.1:3128/"

# This is also a handy place to tweak where Docker's temporary files go.
#export TMPDIR="/mnt/bigdrive/docker-tmp"
EOF
service docker restart

adduser vagrant docker

cd $VAGRANT_HOME/ferry
/usr/local/opt/go/bin/go get github.com/nitrous-io/goop

sudo chown -R vagrant:vagrant $VAGRANT_HOME/go
SCRIPT

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.hostname = 'ferry'
  config.vm.box = 'nitrous-io/trusty64'

  config.vm.provider :virtualbox do |v|
    v.memory = ENV['FERRY_VM_MEM'] || 2048
    v.cpus   = ENV['FERRY_VM_CPU'] || 2
    v.customize ['modifyvm', :id, '--natdnshostresolver1', 'on']
    v.customize ['modifyvm', :id, '--natdnsproxy1', 'on']
  end

  config.vm.provider 'vmware_fusion' do |v|
    v.gui = true
    v.vmx['memsize'] = ENV['FERRY_VM_MEM'] || '2048'
    v.vmx['numvcpus'] = ENV['FERRY_VM_CPU'] || '2'
  end

  if private_ip = ENV['FERRY_INTERFACE']
    config.vm.network :private_network, ip: private_ip
  end

  config.vm.synced_folder './', '/home/vagrant/go/src/github.com/nitrous-io/ferry'

  config.ssh.shell = "bash -c 'BASH_ENV=/etc/profile exec bash'" # avoids 'stdin: is not a tty' error.
  config.ssh.forward_agent = true
  config.ssh.private_key_path = %w(~/.ssh/id_rsa ~/.vagrant.d/insecure_private_key)
  config.vm.provision 'shell', inline: $script
end
