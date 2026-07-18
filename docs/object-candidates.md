# 对象雏形表（object-candidates）

> 左栏 mock 对象；后续接真实 Ontology（06）。

| object_id | display_name | type | key_fields | notes |
| --- | --- | --- | --- | --- |
| OBJ-1001 | 一号产线主机 | Equipment | status, temp_setpoint, runtime_h | 金牌 G01 |
| OBJ-1002 | 二号维保单元 | Equipment | maintenance_cycle_d | 金牌 G03 |
| OBJ-1003 | 标定仪 | Instrument | calib_zero, last_calib | 金牌 G11 |
| LINE-A | A 产线 | Line | upstream, downstream | 关联说明 |
| HYD-01 | 液压站 | Utility | oil_temp_min, oil_temp_max | 金牌 G07 |

最少 5 类；desktop 左栏读 `desktop/src/mocks/objects.json`（与本表同步）。
