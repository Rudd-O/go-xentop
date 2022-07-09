BINDIR=/usr/bin
SYSCONFDIR=/etc
UNITDIR=/usr/lib/systemd/system
DESTDIR=
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

bin/prometheus-xentop: prometheus-xentop/*.go *.go
	cd $(ROOT_DIR) && \
	GOBIN=$(ROOT_DIR)/bin CGO_ENABLED=1 go install ./...

.PHONY: clean dist rpm srpm install

prometheus-xentop.service: prometheus-xentop.service.in
	cd $(ROOT_DIR) && \
	cat prometheus-xentop.service.in | \
	sed "s|@NAME@|prometheus-xentop|" | \
	sed "s|@UNITDIR@|$(UNITDIR)|" | \
	sed "s|@BINDIR@|$(BINDIR)|" | \
	sed "s|@SYSCONFDIR@|$(SYSCONFDIR)|" \
	> prometheus-xentop.service

clean:
	cd $(ROOT_DIR) && find -name '*~' -print0 | xargs -0r rm -fv && rm -fr *.tar.gz *.rpm && rm -rf bin && rm -f *.service

dist: clean
	@which rpmspec || { echo 'rpmspec is not available.  Please install the rpm-build package with the command `dnf install rpm-build` to continue, then rerun this step.' ; exit 1 ; }
	cd $(ROOT_DIR) || exit $$? ; excludefrom= ; test -f .gitignore && excludefrom=--exclude-from=.gitignore ; DIR=`rpmspec -q --queryformat '%{name}-%{version}\n' *spec | head -1` && FILENAME="$$DIR.tar.gz" && tar cvzf "$$FILENAME" --exclude="$$FILENAME" --exclude=.git --exclude=.gitignore $$excludefrom --transform="s|^|$$DIR/|" --show-transformed *

srpm: dist
	@which rpmbuild || { echo 'rpmbuild is not available.  Please install the rpm-build package with the command `dnf install rpm-build` to continue, then rerun this step.' ; exit 1 ; }
	cd $(ROOT_DIR) || exit $$? ; rpmbuild --define "_srcrpmdir ." -ts `rpmspec -q --queryformat '%{name}-%{version}.tar.gz\n' *spec | head -1`

rpm: dist
	@which rpmbuild || { echo 'rpmbuild is not available.  Please install the rpm-build package with the command `dnf install rpm-build` to continue, then rerun this step.' ; exit 1 ; }
	cd $(ROOT_DIR) || exit $$? ; rpmbuild --define "_srcrpmdir ." --define "_rpmdir builddir.rpm" -ta `rpmspec -q --queryformat '%{name}-%{version}.tar.gz\n' *spec | head -1`
	cd $(ROOT_DIR) ; mv -f builddir.rpm/*/* . && rm -rf builddir.rpm

install-prometheus-xentop: bin/prometheus-xentop
	install -Dm 755 bin/prometheus-xentop -t $(DESTDIR)/$(BINDIR)/

install-prometheus-xentop.service: prometheus-xentop.service
	install -Dm 644 prometheus-xentop.service -t $(DESTDIR)/$(UNITDIR)/
	echo Now please systemctl --system daemon-reload >&2

install-prometheus-xentop.default:
	install -Dm 644 prometheus-xentop.default $(DESTDIR)/$(SYSCONFDIR)/default/prometheus-xentop

install: install-prometheus-xentop install-prometheus-xentop.service install-prometheus-xentop.default
