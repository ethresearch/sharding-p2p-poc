#!/usr/bin/env python

import subprocess
import sys
import os 
import time

# --i <image>
# --n <number of nodes>
# --I <interface> eno4 
# example: 
# os.system("./umba --i sharding --I eno4 --n 20")
# os.system("docker exec -it whiteblock-node0 -port=8080")
# ip eample: 10.1.0.2 would be whiteblock-node0
# 10.1.0.6 would be whiteblock-node1 ++

#TEST SERIES A
#LATENCY

#A1
#0ms
print("A1 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000") 


print("A1 Complete")
print("Going To Sleep")
time.sleep(1200)

print("Parsing & Saving Data")

os.system("")

os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x A1")

print("Shutting Down")

#A2
#50ms 
print("A2 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0 --latency 12") 


print("A2 Complete")

print("Going To Sleep")
time.sleep(1800)

print("Parsing & Saving Data")
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x A2")

print("Shutting Down")

#A3
#100ms
print("A3 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0 --latency 25") 

print("A3 Complete")

print("Going To Sleep")
time.sleep(1800)

print("Parsing & Saving Data")
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x A3")

print("Shutting Down")

#TEST SERIES B
#TRANSACTION VOLUME

#B1
#8000 tx
print("B1 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 8000") 
print("B1 Complete")

print("Going To Sleep")
time.sleep(1800)

print("Parsing & Saving Data")
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x B1")

#B2
#16000 tx
print("B2 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 16000") 
print("B2 Complete")

print("Going To Sleep")
time.sleep(1800)

print("Parsing & Saving Data")
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x B2")
print("Shutting Down")

#B3
#20000tx
print("B3 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 20000") 
print("B3 Complete")

print("Going To Sleep")
time.sleep(1800)

print("Parsing & Saving Data")
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x B3")
print("Shutting Down")

#TEST SERIES C
#TRANSACTION SIZE

#C1
#500B
print("C1 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --tx-size 259") 
print("C1 Complete")

print("Going To Sleep")
time.sleep(1200)

print("Parsing & Saving Data")
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x C1")
print("Shutting Down")

#C2
#1000B
print("C2 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --tx-size 344") 
print("C2 Complete")

print("Going To Sleep")
time.sleep(1200)

print("Parsing & Saving Data")
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x C2")
print("Shutting Down")

#C3
#2000B
print("C3 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --tx-size 429") 
print("C3 Complete")

print("Going To Sleep")
time.sleep(1200)

print("Parsing & Saving Data")
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x C3")
print("Shutting Down")

#TEST SERIES D
#HIGHER LATENCY

#D1
#200ms
print("D1 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --latency 50") 
print("D1 Complete")

time.sleep(1800)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x D1")

#D2
#300ms
print("D2")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --latency 75") 


time.sleep(1800)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x D2")


#D3
#400ms
print("D3")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0 --latency 100") 


time.sleep(1800)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x D3")


#TEST SERIES E
#PACKET LOSS

#E1
#0.01%
print("E1")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0.0025 --latency 0") 

time.sleep(1800)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x E1")


#E2
#0.1%
print("E2")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0.025 --latency 0") 


time.sleep(1800)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x E2")


#E3
#1%
print("E3")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0.25 --latency 0") 


time.sleep(1800)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x E3")

#TEST SERIES F
#NUMBER OF BLOCK PRODUCERS

#F1
#3BP
print("F1")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 4  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 4 --clients 24 --number-of-tx 4000") 

time.sleep(1200)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x F1")

#F2
#11BP
print("F2")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 12  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 12 --clients 24 --number-of-tx 4000") 

time.sleep(1200)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x F2")

#F3
#15BP
print("F3")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 16  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 16 --clients 24 --number-of-tx 4000") 

time.sleep(1200)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x F3")

#TEST SERIES G
#HIGHER NUMBER OF CLIENTS

#G1
#100
print("G1")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 100 --number-of-tx 4000") 

time.sleep(1200)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x G1")

#G2
#200
print("G2")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 200 --number-of-tx 4000") 

time.sleep(1200)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x G2")

#G3
#300
print("G3")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 300 --number-of-tx 4000") 

time.sleep(1200)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x G3")

#TEST SERIES H
#H1 DONKEY KONG 
print("H1 Start")
os.chdir("/home/appo/umba")
os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 3000 --number-of-tx 4000 --emulate --loss 0.5 --latency 25") 


time.sleep(7200)
os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x E3")
