; Setzer — Windows installer (NSIS / Modern UI 2)
; Per-user install (no admin / UAC). Built with makensis (runs on Linux/CI).
; Usage: makensis -DVERSION=x.y.z packaging/windows/setzer.nsi

!define APPNAME "Setzer"
!define COMPANY "crux"
!ifndef VERSION
  !define VERSION "0.0.0"
!endif
; EXE (source binary) and OUT (installer path) are passed absolute by `make
; windows`; defaults let the script also build from the repo root directly.
!ifndef EXE
  !define EXE "dist/setzer.exe"
!endif
!ifndef OUT
  !define OUT "dist/Setzer-Setup-${VERSION}.exe"
!endif
!define UNINSTKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}"

Name "${APPNAME}"
OutFile "${OUT}"
Unicode true
RequestExecutionLevel user
InstallDir "$LOCALAPPDATA\Programs\${APPNAME}"

!include "MUI2.nsh"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
; Offer to launch Setzer at the end of the wizard.
!define MUI_FINISHPAGE_RUN "$INSTDIR\setzer.exe"
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File "${EXE}"

  ; Start Menu shortcut — launches Setzer (opens the browser by default).
  CreateDirectory "$SMPROGRAMS\${APPNAME}"
  CreateShortcut "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk" "$INSTDIR\setzer.exe"

  ; Uninstaller + "Settings -> Apps" entry (per-user, HKCU).
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayName" "${APPNAME}"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${UNINSTKEY}" "Publisher" "${COMPANY}"
  WriteRegStr HKCU "${UNINSTKEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${UNINSTKEY}" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegDWORD HKCU "${UNINSTKEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINSTKEY}" "NoRepair" 1
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\setzer.exe"
  Delete "$INSTDIR\Uninstall.exe"
  Delete "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk"
  RMDir "$SMPROGRAMS\${APPNAME}"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "${UNINSTKEY}"
SectionEnd
