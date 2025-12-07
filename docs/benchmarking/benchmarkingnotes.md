_____________________


## Setting Up Profile


Be sure to use a blank import

	_ "net/http/pprof" // Blank import to register pprof handlers

Create the server

	go func() {
		fmt.Println("Running server")
		http.ListenAndServe("localhost:6060", nil)
	}()

	runtime.SetCPUProfileRate(1000)
	runtime.MemProfileRate = 1

	g := NewGame()

	g.gameUI.CreateMainInterface(&g.playerData, &g.em)

	ebiten.SetWindowResizable(true)

	ebiten.SetWindowTitle("Tower")

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}


## Not sure what this is for anymore


_____________________
Open the pprof URL in a browser:
bash
Copy code
http://localhost:6060/debug/pprof/profile?seconds=30
This command collects a CPU profile for 30 seconds. Once the profile is captured, a file named profile.pb.gz will be downloaded.


______________________

# USAGE

## Collecting a Profile Through the terminal


### In the terminal window, type the following


curl -o cpu_profile.pb.gz http://localhost:6060/debug/pprof/profile?seconds=60

### Then run the following command to view the profile

go tool pprof cpu_profile.pb.gz


___

## Viewing Top Processes

---

Top:
Shows the top hot spots in terms of CPU time.


bash
Copy code
(pprof) top
Output will show which functions are using the most CPU time:

bash
Copy code
Showing nodes accounting for 80ms, 80% of 100ms total
flat  flat%   sum%        cum   cum%
50ms  50.00%  50.00%      50ms  50.00%  mypackage.myFunc
30ms  30.00%  80.00%      80ms  80.00%  mypackage.anotherFunc
Top N:


## Creating Graph
---

(pprof) web for graph

The following seems to do the same:

svg > outputfilename.svg
---
Or t


## Tree Representation  Calls In Terminal

tree

tree callers. I.E, tree runtime.tracebackPCs

### Getting graphs of specific callers

I.E, 

web -node=runtime.systemstack


## Getting Graph of 

## Listing Callers

pprof list yourpkg.SomeFunction