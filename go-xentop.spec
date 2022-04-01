%define debug_package %{nil}

%define mybuildnumber %{?build_number}%{?!build_number:1}

Name:           go-xentop
Version:        1.0.2
Release:        %{mybuildnumber}%{?dist}
Summary:        Wraps xentop.
Group:          Applications/System

License:        LGPLv3
URL:            https://github.com/Rudd-O/go-xentop
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  sed
BuildRequires:  make
BuildRequires:  golang

%description
This package runs a Prometheus exporter that exports Xen VM statistics.

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
%config(noreplace) %{_sysconfdir}/default/prometheus-xentop
%{_unitdir}/prometheus-xentop.service
%attr(0755, root, root) %{_bindir}/*
%doc %{_defaultdocdir}/%{name}/README.md

%post
%systemd_post prometheus-xentop.service

%preun
%systemd_preun prometheus-xentop.service

%postun
%systemd_postun_with_restart prometheus-xentop.service

%changelog
* Tue Oct 19 2021  Manuel Amador (Rudd-O) <rudd-o@rudd-o.com>
- Add generic pipeline build.
