%define debug_package %{nil}

%define mybuildnumber %{?build_number}%{?!build_number:1}

Name:           prometheus-xentop
Version:        2.0.9
Release:        %{mybuildnumber}%{?dist}
Summary:        Prometheus exporter for xen stats.
Group:          Applications/System

License:        LGPLv3
URL:            https://github.com/Rudd-O/prometheus-xentop
Source0:        %{name}-%{version}.tar.gz

Obsoletes:      go-xentop < 2.0.1
BuildRequires:  sed
BuildRequires:  make
BuildRequires:  golang
BuildRequires:  systemd-rpm-macros
BuildRequires:  xen-devel

%description
This package runs a Prometheus exporter that exports Xen VM statistics.
This new major version changes metric names and uses direct connection
to xend for more accurate statistics, and statistics not collected
before, like RAM usage.

%prep
%setup -q

%build
%{make_build} UNITDIR=%{_unitdir} BINDIR=%{_bindir} SYSCONFDIR=%{_sysconfdir}

%install
%{make_install} DESTDIR="%{buildroot}" UNITDIR=%{_unitdir} BINDIR=%{_bindir} SYSCONFDIR=%{_sysconfdir}
mkdir -p "%{buildroot}%{_defaultdocdir}/%{name}"
cp -f "README.md" "%{buildroot}%{_defaultdocdir}/%{name}/README.md"

%files
%defattr(-, root, root)
%config(noreplace) %{_sysconfdir}/default/%{name}
%{_unitdir}/%{name}.service
%attr(0755, root, root) %{_bindir}/*
%doc %{_defaultdocdir}/%{name}/README.md

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%changelog
* Tue Oct 19 2021  Manuel Amador (Rudd-O) <rudd-o@rudd-o.com>
- Add generic pipeline build.
