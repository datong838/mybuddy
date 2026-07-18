; OpenOcta Windows Installer (NSIS)
; - Installs openocta.exe and openocta-launcher.exe
; - Creates Start Menu + Desktop shortcuts
; - Registers auto-start (current user)

Unicode true

!include "MUI2.nsh"

Name "OpenOcta"
OutFile "OpenOcta-Setup.exe"
InstallDir "$PROGRAMFILES64\\OpenOcta"
InstallDirRegKey HKLM "Software\\OpenOcta" "InstallDir"
RequestExecutionLevel admin

!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "license.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "SimpChinese"

Section "OpenOcta" SEC01
  SetOutPath "$INSTDIR"

  ; Binaries (to be provided next to this .nsi when building)
  File "openocta.exe"
  File "openocta-launcher.exe"

  ; 创建数据目录（配置、日志等），launcher 会设置 OPENOCTA_STATE_DIR=$INSTDIR\data
  CreateDirectory "$INSTDIR\\data"

  ; Uninstaller
  WriteUninstaller "$INSTDIR\\Uninstall.exe"

  ; Registry install dir（launcher 读取此路径并设置 OPENOCTA_STATE_DIR）
  WriteRegStr HKLM "Software\\OpenOcta" "InstallDir" "$INSTDIR"

  ; Start Menu
  CreateDirectory "$SMPROGRAMS\\OpenOcta"
  CreateShortCut "$SMPROGRAMS\\OpenOcta\\OpenOcta.lnk" "$INSTDIR\\openocta-launcher.exe"
  CreateShortCut "$SMPROGRAMS\\OpenOcta\\卸载 OpenOcta.lnk" "$INSTDIR\\Uninstall.exe"

  ; Desktop shortcut
  CreateShortCut "$DESKTOP\\OpenOcta.lnk" "$INSTDIR\\openocta-launcher.exe"

  ; Auto-start (current user)
  WriteRegStr HKCU "Software\\Microsoft\\Windows\\CurrentVersion\\Run" "OpenOcta" "$INSTDIR\\openocta-launcher.exe"
SectionEnd

Section "Uninstall"
  ; Remove auto-start
  DeleteRegValue HKCU "Software\\Microsoft\\Windows\\CurrentVersion\\Run" "OpenOcta"

  ; Shortcuts
  Delete "$DESKTOP\\OpenOcta.lnk"
  Delete "$SMPROGRAMS\\OpenOcta\\OpenOcta.lnk"
  Delete "$SMPROGRAMS\\OpenOcta\\卸载 OpenOcta.lnk"
  RMDir "$SMPROGRAMS\\OpenOcta"

  ; Files
  Delete "$INSTDIR\\openocta.exe"
  Delete "$INSTDIR\\openocta-launcher.exe"
  Delete "$INSTDIR\\Uninstall.exe"

  ; Registry
  DeleteRegKey HKLM "Software\\OpenOcta"

  ; Dir
  RMDir "$INSTDIR"
SectionEnd

