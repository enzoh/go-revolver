/**
 * File        : neighbors.go
 * Description : Sorting algorithm.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"crypto/sha256"
	"math"
	"sort"

	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

type keyref struct {
	x   float64
	y   float64
	z   float64
	ref int
}

var ANGLES = [32]float64{
	2.454369260617026e-2,
	9.587379924285257e-5,
	3.7450702829239286e-7,
	1.4629180792671596e-9,
	5.714523747137342e-12,
	2.2322358387255243e-14,
	8.71967124502158e-17,
	3.4061215800865545e-19,
	1.3305162422213103e-21,
	5.1973290711769935e-24,
	2.030206668428513e-26,
	7.930494798548879e-29,
	3.097849530683156e-31,
	1.2100974729231078e-33,
	4.72694325360589e-36,
	1.8464622084398007e-38,
	7.212743001717972e-41,
	2.8174777350460826e-43,
	1.100577240252376e-45,
	4.299129844735844e-48,
	1.679347595599939e-50,
	6.559951545312262e-53,
	2.5624810723876022e-55,
	1.0009691689014071e-57,
	3.9100358160211216e-60,
	1.5273577406332506e-62,
	5.966241174348635e-65,
	2.3305629587299356e-67,
	9.103761557538811e-70,
	3.556156858413598e-72,
	1.3891237728178117e-74,
	5.426264737569577e-77,
}

var HEIGHTS = [32]float64{
	3.90625e-3,
	1.52587890625e-5,
	5.960464477539063e-8,
	2.3283064365386963e-10,
	9.094947017729282e-13,
	3.552713678800501e-15,
	1.3877787807814457e-17,
	5.421010862427522e-20,
	2.117582368135751e-22,
	8.271806125530277e-25,
	3.2311742677852644e-27,
	1.262177448353619e-29,
	4.930380657631324e-32,
	1.925929944387236e-34,
	7.52316384526264e-37,
	2.938735877055719e-39,
	1.1479437019748901e-41,
	4.484155085839415e-44,
	1.7516230804060213e-46,
	6.842277657836021e-49,
	2.6727647100921956e-51,
	1.044048714879764e-53,
	4.078315292499078e-56,
	1.5930919111324523e-58,
	6.223015277861142e-61,
	2.4308653429145085e-63,
	9.495567745759799e-66,
	3.7092061506874214e-68,
	1.448908652612274e-70,
	5.659799424266695e-73,
	2.210859150104178e-75,
	8.636168555094445e-78,
}

//
func coordinates(id peer.ID) (float64, float64) {

	var alpha, beta float64

	h1 := sha256.Sum256([]byte(id))

	for i := 0; i < 32; i++ {
		alpha += ANGLES[i] * float64(h1[i])
	}

	h2 := sha256.Sum256(h1[:])

	for i := 0; i < 32; i++ {
		beta += HEIGHTS[i] * float64(h2[i])
	}

	beta = math.Pi - math.Acos(2*beta-1)

	return alpha, beta

}

//
func convert(alpha, beta float64) (float64, float64, float64) {

	x := math.Cos(alpha) * math.Sin(beta)
	y := math.Sin(alpha) * math.Sin(beta)
	z := math.Cos(beta)

	return x, y, z

}

//
func points(target peer.ID, peers []peer.ID) []keyref {

	theta, phi := coordinates(target)

	rotate := func(alpha float64, beta float64) (float64, float64) {
		return alpha + math.Pi - theta, beta + math.Pi/2 - phi
	}

	var points []keyref

	for i := range peers {

		if peers[i] == target {
			continue
		}

		alpha, beta := coordinates(peers[i])

		gamma, delta := rotate(alpha, beta)

		x, y, z := convert(gamma, delta)

		j := sort.Search(len(points), func(k int) bool {
			return x < points[k].x
		})

		points = append(points, keyref{})
		copy(points[j+1:], points[j:])
		points[j] = keyref{x, y, z, i}

	}

	return points

}

//
func balance(xs, ys []keyref) ([]keyref, []keyref) {

	if len(xs) < len(ys) {

		n := (len(ys) - len(xs)) / 2

		for i := len(ys) - 1; i >= len(ys)-n; i-- {
			xs = append(xs, ys[i])
		}

		ys = ys[0 : len(ys)-n]

	} else if len(ys) < len(xs) {

		n := (len(xs) - len(ys)) / 2

		for i := len(xs) - 1; i >= len(xs)-n; i-- {
			ys = append(ys, xs[i])
		}

		xs = xs[0 : len(xs)-n]

	}

	return xs, ys

}

//
func closest(target peer.ID, peers []peer.ID) []peer.ID {

	points := points(target, peers)

	res := make([]peer.ID, len(points))
	for i := range res {
		res[i] = peers[points[i].ref]
	}

	return res

}

//
func neighbors(target peer.ID, peers []peer.ID) ([]peer.ID, []peer.ID, []peer.ID, []peer.ID) {

	points := points(target, peers)

	var pointsE, pointsW, pointsNE, pointsSE, pointsSW, pointsNW []keyref

	for i := range points {
		if points[i].y < 0 {
			pointsE = append(pointsE, points[i])
		} else {
			pointsW = append(pointsW, points[i])
		}
	}

	pointsE, pointsW = balance(pointsE, pointsW)

	for i := range pointsE {
		if pointsE[i].z < 0 {
			pointsSE = append(pointsSE, pointsE[i])
		} else {
			pointsNE = append(pointsNE, pointsE[i])
		}
	}

	for i := range pointsW {
		if pointsW[i].z < 0 {
			pointsSW = append(pointsSW, pointsW[i])
		} else {
			pointsNW = append(pointsNW, pointsW[i])
		}
	}

	pointsNE, pointsSE = balance(pointsNE, pointsSE)
	pointsNW, pointsSW = balance(pointsNW, pointsSW)

	resNE := make([]peer.ID, len(pointsNE))
	for i := range pointsNE {
		resNE[i] = peers[pointsNE[i].ref]
	}

	resSE := make([]peer.ID, len(pointsSE))
	for i := range pointsSE {
		resSE[i] = peers[pointsSE[i].ref]
	}

	resSW := make([]peer.ID, len(pointsSW))
	for i := range pointsSW {
		resSW[i] = peers[pointsSW[i].ref]
	}

	resNW := make([]peer.ID, len(pointsNW))
	for i := range pointsNW {
		resNW[i] = peers[pointsNW[i].ref]
	}

	return resNE, resSE, resSW, resNW

}
