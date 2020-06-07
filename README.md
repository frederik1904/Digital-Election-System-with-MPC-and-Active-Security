# Digital-Election-System-with-MPC-and-Active-Security
## What is the project?
This project was made for our bachelor degree at Aarhus university. 
The project contains the Golang code for making a digital online election with privacy and integrity of the result.


## How to run it:
To run the code you need to have Golang installed (https://golang.org/doc/install). When this is installed you can either run make to install all neded plugins, and then run the code. You can also install the needed libaries yourself (the needed libaries can be found in the make file) and run the code with "go run main.go". 

For the algorithms to work, there is a need of atleast 3 servers, and if it is run locally the debug value can be set to 1 or above to make it automaticly use the local ip-address 127.0.0.1:8080.

To run the software you need to start all your servers, and then from the first server press any button to start "Phase Two" which is when they have exchanged the needed primes and public keys to compute votes.

## Iterations
The other iterations can be found as seperate brances, where the "Passive with flooding" only secures that the votes are private, and the "Active verification test" secures that the specific vote evaluates to either 1 or 0 but does not necesarily stem from the same polynomial. In the master branch we have implemented Non-interactive Secret sharing scheme and Zeroknowledge proofs to ensure that all points from a vote comes from the same polynomial and removes the need for flooding a verification share on the network.
