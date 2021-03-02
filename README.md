Generate Image with GA
====

Growth
----

![Growth](https://raw.githubusercontent.com/technohippy/go-gaimage/main/images/cat200_growth.gif)

### Config

```
ResultsDir = "results"
RestoreFromDump = false
LogStride = 100
UseAlpha = false
UseGeneMutate = true
UseTournament = false
RunSeparately = false
GenrationCount = 5000
PopulationCount = 40
EliteCount = 10
TournamentCount = 2
GeneCount = 300
LocusCount = 7
MutateRatio = 0.5
MutateProbability = 0.2
ShapeSizeMin = 4
ShapeSizeMax = 30
```

Result
----

![Target](https://raw.githubusercontent.com/technohippy/go-gaimage/main/images/cat200.png)
![Generated](https://raw.githubusercontent.com/technohippy/go-gaimage/main/images/cat200_result.png)

### Config

```
ResultsDir = "results"
RestoreFromDump = false
LogStride = 100
UseAlpha = false
UseGeneMutate = true
UseTournament = false
RunSeparately = true
GenrationCount = 50000
PopulationCount = 40
EliteCount = 10
TournamentCount = 2
GeneCount = 300
LocusCount = 7
MutateRatio = 0.5
MutateProbability = 0.2
ShapeSizeMin = 4
ShapeSizeMax = 30
```

Refs.
----

- https://gamingchahan.com/ecchi/
- https://kennycason.com/posts/2017-10-01-genetic-algorithm-draw-images-japanese.html