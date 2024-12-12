NAME=prometheus-xentop
BINDIR=/usr/bin
SYSCONFDIR=/etc
UNITDIR=/usr/lib/systemd/system
DESTDIR=
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

bin/$(NAME): xenstat/*.go cmd/$(NAME)/*.go
	cd $(ROOT_DIR) && \
	GOBIN=$(ROOT_DIR)/bin CGO_ENABLED=1 go install $(BUILD_PKG_EXTRA_GO_FLAGS) ./...


.PHONY: clean dist rpm srpm install

$(NAME).service: $(NAME).service.in
	cd $(ROOT_DIR) && \
	cat $(NAME).service.in | \
	sed "s|@NAME@|$(NAME)|" | \
	sed "s|@UNITDIR@|$(UNITDIR)|" | \
	sed "s|@BINDIR@|$(BINDIR)|" | \
	sed "s|@SYSCONFDIR@|$(SYSCONFDIR)|" \
	> $(NAME).service

clean:
	cd $(ROOT_DIR) && find -name '*~' -print0 | xargs -0r rm -fv && rm -fr *.tar.gz *.rpm && rm -rf bin && rm -f *.service
	rm -rf debian/README.md debian/$(NAME).service

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

deb:
	debuild -us -uc

install-$(NAME): bin/$(NAME)
	install -Dm 755 bin/$(NAME) -t $(DESTDIR)/$(BINDIR)/

ifndef DEB_BUILD_ARCH
install-$(NAME).service: $(NAME).service
	install -Dm 644 $(NAME).service -t $(DESTDIR)/$(UNITDIR)/
	echo Now please systemctl --system daemon-reload >&2
else
# Debian package builds: debhelper automatically enables systemd services for service files in debian/
install-$(NAME).service: $(NAME).service
	cp $(NAME).service -t debian/
	cp README.md -t debian/
endif

install-$(NAME).default:
	install -Dm 644 $(NAME).default $(DESTDIR)/$(SYSCONFDIR)/default/$(NAME)

install: install-$(NAME) install-$(NAME).service install-$(NAME).default
