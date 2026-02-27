#!/bin/bash
# InfluxDB 初始化脚本

echo "Initializing InfluxDB..."

# 创建数据库
influx -execute "CREATE DATABASE IF NOT EXISTS stock_market"

# 创建保留策略（可选）
# 保留原始数据2年，聚合数据5年
influx -database stock_market -execute "CREATE RETENTION POLICY two_years ON stock_market DURATION 104w REPLICATION 1 DEFAULT"
influx -database stock_market -execute "CREATE RETENTION POLICY five_years ON stock_market DURATION 260w REPLICATION 1"

echo "InfluxDB initialization completed!"

# 查看创建的数据库
influx -execute "SHOW DATABASES"
