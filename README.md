# Digital-Election-System-with-MPC-and-Active-Security
## What is the project?
This project was made for our bachelor degree at Aarhus university. 
The project contains code that makes holding an online vote possible with anonymity and correctness of the outcome.


## How to run it:
To run the code you need to have Golang installed (https://golang.org/doc/install), when this is installed you can either run make to install all neded plugins and run the code or you can install the needed libaries yourself (the needed libaries can be found in the make file) and run the code with "go run main.go". 

To run the software you need to start all your servers, and then from the first server that was started press any button to start "Phase Two" which is when they have exchanged the needed primes to compute votes.

For the algorithems used to work there is a need of atleast 3 servers, and if it is run locally the debug value can be set to 1 or above to make it automaticly know which ip and port to connect to.


## Iterations
The other iterations can be found as seperate brances, where the "Passive with flooding" only secures that the votes are anonyme, and the "Active verification test" secures that the specific vote evaluates to either 1 or 0 but does not necesarily stem from the same polynomial. In the master branch we have implemented NVSS and Zeroknowledge proofs that ensures that all point from a vote comes from the same polynomial and removes the need for flooding the verification share over the network.