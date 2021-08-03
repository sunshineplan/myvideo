#! /bin/bash

installSoftware() {
    apt -qq -y install nginx
}

installMyVideo() {
    mkdir -p /var/www/myvideo
    curl -Lo- https://github.com/sunshineplan/myvideo/releases/latest/download/release.tar.gz | tar zxC /var/www/myvideo
    cd /var/www/myvideo
    chmod +x myvideo
}

configMyVideo() {
    read -p 'Please enter API address: ' api
    read -p 'Please enter unix socket(default: /run/myvideo.sock): ' unix
    [ -z $unix ] && unix=/run/myvideo.sock
    read -p 'Please enter host(default: 127.0.0.1): ' host
    [ -z $host ] && host=127.0.0.1
    read -p 'Please enter port(default: 12345): ' port
    [ -z $port ] && port=12345
    read -p 'Please enter log path(default: /var/log/app/myvideo.log): ' log
    [ -z $log ] && log=/var/log/app/myvideo.log
    read -p 'Please enter update URL: ' update
    read -p 'Please enter exclude files: ' exclude
    mkdir -p $(dirname $log)
    sed "s,\$api,$api," /var/www/myvideo/config.ini.default > /var/www/myvideo/config.ini
    sed -i "s,\$unix,$unix," /var/www/myvideo/config.ini
    sed -i "s,\$log,$log," /var/www/myvideo/config.ini
    sed -i "s/\$host/$host/" /var/www/myvideo/config.ini
    sed -i "s/\$port/$port/" /var/www/myvideo/config.ini
    sed -i "s,\$update,$update," /var/www/myvideo/config.ini
    sed -i "s|\$exclude|$exclude|" /var/www/myvideo/config.ini
    ./myvideo install || exit 1
    service myvideo start
}

writeLogrotateScrip() {
    if [ ! -f '/etc/logrotate.d/app' ]; then
	cat >/etc/logrotate.d/app <<-EOF
		/var/log/app/*.log {
		    copytruncate
		    rotate 12
		    compress
		    delaycompress
		    missingok
		    notifempty
		}
		EOF
    fi
}

setupNGINX() {
    cp -s /var/www/myvideo/scripts/myvideo.conf /etc/nginx/conf.d
    sed -i "s/\$domain/$domain/" /var/www/myvideo/scripts/myvideo.conf
    sed -i "s,\$unix,$unix," /var/www/myvideo/scripts/myvideo.conf
    service nginx reload
}

main() {
    read -p 'Please enter domain:' domain
    installSoftware
    installMyVideo
    configMyVideo
    writeLogrotateScrip
    setupNGINX
}

main
