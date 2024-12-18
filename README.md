# ParaBFT

A novel high-performance Parallel Byzantine Fault Tolerance consensus algorithm.
This codebase is to implement and evaluate the performance of ParaBFT.
This project is based on Bamboo (https://github.com/gitferry/bamboo.git).

## Implementation of ParaBFT
- Parabft modifies the replica.go file based on the original Bamboo implementation, 
  removing the feature where a new leader is selected for each round of consensus.

- We introduces a new file parabft.go, which implements the functionality of allowing multiple
  leaders to package transactions and perform consensus simultaneously.

## Usage
The experiment code needs to be conducted in Ubuntu 20 environment.

1. Navigate to the parabft/bin folder and run ./build.sh to compile the code.
2. After compilation, execute ./total_run.sh to run the program. Once it is running, you can monitor throughput and latency in real-time via the browser at 127.0.0.1:8070/query.
3. After the experiment is completed, use ./bothstop.sh to stop the program.

## Notes
Experiment-related parameters can be configured in config.json, such as:

- Number of nodes (by modifying the number of IP entries),
- Transaction sending rate (Throttle),
- Transaction size (payload_size),
- Number of transactions per block (bsize).