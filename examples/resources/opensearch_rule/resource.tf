# Create a custom security analytics rule
resource "opensearch_rule" "moriya_rootkit" {
  category = "windows"
  rule     = <<EOF
title: Moriya Rootkit
id: 25b9c01c-350d-4b95-bed1-836d04a4f324
description: Detects the use of Moriya rootkit as described in the securelist's Operation TunnelSnake report
status: experimental
author: Bhabesh Raj
date: 2021/05/06
modified: 2021/11/30
references:
    - https://securelist.com/operation-tunnelsnake-and-moriya-rootkit/101831
tags:
    - attack.persistence
    - attack.privilege_escalation
    - attack.t1543.003
logsource:
    product: windows
    service: system
detection:
    selection:
        Provider_Name: 'Service Control Manager'
        EventID: 7045
        ServiceName: ZzNetSvc
    condition: selection
level: critical
falsepositives:
    - Unknown
EOF
}

# Example with forced update/delete
resource "opensearch_rule" "suspicious_process" {
  category = "windows"
  forced   = true # Required if rule is actively used by detectors
  rule     = <<EOF
title: Suspicious Process Execution
id: a1b2c3d4-5e6f-7a8b-9c0d-1e2f3a4b5c6d
description: Detects execution of suspicious processes
status: stable
author: Security Team
date: 2024/01/01
logsource:
    category: process_creation
    product: windows
detection:
    selection:
        Image|endswith:
            - '\suspicious.exe'
            - '\malware.exe'
    condition: selection
level: high
falsepositives:
    - Legitimate software with similar names
EOF
}