namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminGraph {
			admin {
				allSubgraphs {
					nodes {
						name
						color
					}
				}
				allLatestNoteViews {
					nodes {
						id
						subgraphNames
						title
						pathId
						free
						isHomePage
						graphPosition{
							x,
							y,
						}
						inLinks {
							title
							pathId
							id
						}
					}
				}
			}
		}
`)

	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation UpdateNoteGraphPositions($input: UpdateNoteGraphPositionsInput!) {
			admin {
				payload: updateNoteGraphPositions(input: $input) {
					__typename
					... on UpdateNoteGraphPositionsPayload {
						success
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	type graph_layout_node = {
		id: string
		subgraphNames: readonly string[]
		free: boolean
		saved: { x: number, y: number } | null
	}

	type graph_layout_edge = {
		source: string
		target: string
	}

	// Deterministic Fruchterman-Reingold force layout:
	// - repulsion between all node pairs, spring attraction along edges
	// - attraction to subgraph centroids (visible clusters)
	// - optional radial pull of public (free) notes to an outer ring
	// Nodes with saved positions are kept fixed when pin_saved is set.
	function graph_layout_positions(
		nodes: readonly graph_layout_node[],
		edges: readonly graph_layout_edge[],
		opts: { pin_saved: boolean, public_outside: boolean },
	): Map<string, { x: number, y: number }> {

		const result = new Map<string, { x: number, y: number }>()
		const count = nodes.length
		if( !count ) return result

		const L = 150 // ideal edge length
		const scale = L * Math.sqrt( count )
		// O(n^2) per iteration - fewer iterations for big graphs
		const iterations = Math.max( 60, Math.min( 300, Math.floor( 48_000_000 / ( count * count + 1 ) ) ) )

		const index = new Map<string, number>()
		nodes.forEach( ( node, i ) => index.set( node.id, i ) )

		// Seed unsaved nodes on a golden-angle spiral around the saved centroid,
		// so no two nodes ever coincide and the layout is reproducible.
		let cx0 = 0, cy0 = 0, saved_count = 0
		for( const node of nodes ) {
			if( !node.saved ) continue
			cx0 += node.saved.x
			cy0 += node.saved.y
			saved_count++
		}
		if( saved_count ) {
			cx0 /= saved_count
			cy0 /= saved_count
		}

		const xs = new Float64Array( count )
		const ys = new Float64Array( count )
		const pinned = new Uint8Array( count )
		const golden = Math.PI * ( 3 - Math.sqrt( 5 ) )
		let spiral = 0
		nodes.forEach( ( node, i ) => {
			if( node.saved ) {
				xs[ i ] = node.saved.x
				ys[ i ] = node.saved.y
				if( opts.pin_saved ) pinned[ i ] = 1
			} else {
				const radius = L * 0.6 * Math.sqrt( spiral + 1 )
				const angle = spiral * golden
				xs[ i ] = cx0 + radius * Math.cos( angle )
				ys[ i ] = cy0 + radius * Math.sin( angle )
				spiral++
			}
		} )

		const edge_pairs: [ number, number ][] = []
		for( const edge of edges ) {
			const a = index.get( edge.source )
			const b = index.get( edge.target )
			if( a === undefined || b === undefined || a === b ) continue
			edge_pairs.push( [ a, b ] )
		}

		const clusters = new Map<string, number[]>()
		nodes.forEach( ( node, i ) => {
			for( const name of node.subgraphNames ) {
				let members = clusters.get( name )
				if( !members ) clusters.set( name, members = [] )
				members.push( i )
			}
		} )

		const K_CLUSTER = L * 3 // subgraph centroid springs, tighter than global scale
		const K_GRAVITY = L * 12 // weak pull to center, keeps components together
		const R_PUBLIC = scale * 0.6 // target ring radius for public notes

		const dx = new Float64Array( count )
		const dy = new Float64Array( count )
		let temp = scale / 4
		const cool = ( temp - 1 ) / iterations

		for( let iter = 0; iter < iterations; iter++ ) {
			dx.fill( 0 )
			dy.fill( 0 )

			// pairwise repulsion: f = L^2 / d
			for( let i = 0; i < count; i++ ) {
				for( let j = i + 1; j < count; j++ ) {
					let ddx = xs[ i ] - xs[ j ]
					let ddy = ys[ i ] - ys[ j ]
					let d2 = ddx * ddx + ddy * ddy
					if( d2 < 1e-4 ) {
						// deterministic jitter for coincident nodes
						ddx = ( ( ( i * 37 + j * 11 ) % 17 ) - 8 ) * 0.01 || 0.01
						ddy = ( ( ( i * 13 + j * 29 ) % 15 ) - 7 ) * 0.01 || 0.01
						d2 = ddx * ddx + ddy * ddy
					}
					const dist = Math.sqrt( d2 )
					const force = L * L / dist
					const fx = ddx / dist * force
					const fy = ddy / dist * force
					dx[ i ] += fx
					dy[ i ] += fy
					dx[ j ] -= fx
					dy[ j ] -= fy
				}
			}

			// spring attraction along edges: f = d^2 / L
			for( const [ a, b ] of edge_pairs ) {
				const ddx = xs[ a ] - xs[ b ]
				const ddy = ys[ a ] - ys[ b ]
				const dist = Math.sqrt( ddx * ddx + ddy * ddy ) || 0.01
				const force = dist * dist / L
				const fx = ddx / dist * force
				const fy = ddy / dist * force
				dx[ a ] -= fx
				dy[ a ] -= fy
				dx[ b ] += fx
				dy[ b ] += fy
			}

			// attraction to subgraph centroids: clusters emerge
			for( const members of clusters.values() ) {
				if( members.length < 2 ) continue
				let mx = 0, my = 0
				for( const i of members ) {
					mx += xs[ i ]
					my += ys[ i ]
				}
				mx /= members.length
				my /= members.length
				for( const i of members ) {
					const ddx = mx - xs[ i ]
					const ddy = my - ys[ i ]
					const dist = Math.sqrt( ddx * ddx + ddy * ddy ) || 0.01
					const force = dist * dist / K_CLUSTER
					dx[ i ] += ddx / dist * force
					dy[ i ] += ddy / dist * force
				}
			}

			// layout centroid for gravity and the public ring
			let cx = 0, cy = 0
			for( let i = 0; i < count; i++ ) {
				cx += xs[ i ]
				cy += ys[ i ]
			}
			cx /= count
			cy /= count

			for( let i = 0; i < count; i++ ) {
				const ddx = cx - xs[ i ]
				const ddy = cy - ys[ i ]
				const dist = Math.sqrt( ddx * ddx + ddy * ddy ) || 0.01

				// weak gravity to center
				const force = dist * dist / K_GRAVITY
				dx[ i ] += ddx / dist * force
				dy[ i ] += ddy / dist * force

				// radial spring pulling public notes to the outer ring
				if( opts.public_outside && nodes[ i ].free ) {
					const out = ( R_PUBLIC - dist ) * 0.3
					dx[ i ] -= ddx / dist * out
					dy[ i ] -= ddy / dist * out
				}
			}

			// apply displacements, capped by temperature
			for( let i = 0; i < count; i++ ) {
				if( pinned[ i ] ) continue
				const len = Math.sqrt( dx[ i ] * dx[ i ] + dy[ i ] * dy[ i ] )
				if( len < 1e-9 ) continue
				const step = Math.min( len, temp )
				xs[ i ] += dx[ i ] / len * step
				ys[ i ] += dy[ i ] / len * step
			}

			temp -= cool
		}

		nodes.forEach( ( node, i ) => {
			result.set( node.id, { x: xs[ i ], y: ys[ i ] } )
		} )

		return result
	}

	export class $trip2g_admin_noteview_graph extends $.$trip2g_admin_noteview_graph {
		auto_layout() {
			this.Graph().relayout()
		}

		save_positions() {
			this.Graph().save_all()
		}
	}

	export class $trip2g_admin_noteview_graph_cytoscape extends $.$trip2g_admin_noteview_graph_cytoscape {
		static cytoscape(): any {
			return $mol_import.script( 'https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.32.0/cytoscape.min.js' ).cytoscape
		}

		cytoscape(): any {
			return $trip2g_admin_noteview_graph_cytoscape.cytoscape()
		}

		@$mol_mem
		data() {
			const res = request()

			return {
				nodes: res.admin.allLatestNoteViews.nodes,
				subgraphs: res.admin.allSubgraphs.nodes,
			}
		}

		@$mol_mem
		graph_data() {
			const nodeFilter = ( item: any ) => !item.id.includes( 'sidebar' )
			const { nodes, subgraphs } = this.data()
			const data = nodes.filter( nodeFilter ).map( item => ( {
				...item,
				inLinks: item.inLinks.filter( nodeFilter ),
			} ) )

			return { data, subgraphs }
		}

		layout_input( use_saved: boolean ) {
			const { data } = this.graph_data()

			const nodes: graph_layout_node[] = data.map( item => ( {
				id: item.id,
				subgraphNames: item.subgraphNames ?? [],
				free: Boolean( item.free ),
				saved: use_saved && item.graphPosition
					? { x: item.graphPosition.x, y: item.graphPosition.y }
					: null,
			} ) )

			const edges: graph_layout_edge[] = []
			for( const item of data ) {
				for( const inLink of item.inLinks ) {
					edges.push( { source: inLink.id, target: item.id } )
				}
			}

			return { nodes, edges }
		}

		@$mol_mem
		cytoscape_instance() {
			return this.cytoscape()( {
				container: this.dom_node()
			} )
		}

		render() {
			const cy = this.cytoscape_instance()
			const { data, subgraphs } = this.graph_data()

			// Clear all existing elements from the graph
			cy.elements().remove()

			// Create subgraph color mapping
			const subgraphColors = new Map()
			for( const subgraph of subgraphs ) {
				subgraphColors.set( subgraph.name, subgraph.color || '#666' )
			}

			// Helper function to get node color from first subgraph or free status
			const getNodeColor = ( nodeData: any ) => {
				// Free notes are always green
				if( nodeData.free ) {
					return '#00ff00'
				}

				if( nodeData.subgraphNames && nodeData.subgraphNames.length > 0 ) {
					const firstSubgraph = nodeData.subgraphNames[ 0 ]
					return subgraphColors.get( firstSubgraph ) || '#666'
				}
				return '#666'
			}

			// Helper function to get node shape
			const getNodeShape = ( nodeData: any ) => {
				// Home page notes are diamonds
				if( nodeData.isHomePage ) {
					return 'diamond'
				}
				// Free notes are squares
				if( nodeData.free ) {
					return 'square'
				}
				return 'ellipse'
			}

			// Nodes without a saved position would all land at cytoscape's default
			// (0,0). Lay them out around the pinned (saved) ones instead.
			const { nodes: layout_nodes, edges: layout_edges } = this.layout_input( true )
			const need_layout = layout_nodes.some( node => !node.saved )
			const positions = need_layout
				? graph_layout_positions( layout_nodes, layout_edges, {
					pin_saved: true,
					public_outside: false,
				} )
				: null

			// Prepare nodes and edges from the data
			const elements = []

			// Add all nodes first
			for( const node of data ) {
				const element: any = {
					data: {
						id: node.id,
						label: node.title,
						pathId: node.pathId,
						color: getNodeColor( node ),
						shape: getNodeShape( node )
					}
				}

				const position = node.graphPosition ?? positions?.get( node.id )
				if( position ) {
					element.position = { x: position.x, y: position.y }
				}

				elements.push( element )
			}

			// Add edges
			for( const node of data ) {
				for( const inLink of node.inLinks ) {
					elements.push( {
						data: {
							id: `${ inLink.id }-${ node.id }`,
							source: inLink.id,
							target: node.id
						}
					} )
				}
			}

			// Add elements to cytoscape
			cy.add( elements )

			// Apply styling
			cy.style( [
				{
					selector: 'node',
					style: {
						'background-color': 'data(color)',
						'shape': 'data(shape)',
						'label': 'data(label)',
						'text-valign': 'center',
						'text-halign': 'center',
						'font-size': '12px',
						'color': '#fff',
						'text-outline-width': 2,
						'text-outline-color': '#000'
					}
				},
				{
					selector: 'edge',
					style: {
						'width': 2,
						'line-color': '#ccc',
						'target-arrow-color': '#ccc',
						'target-arrow-shape': 'triangle',
						'curve-style': 'bezier'
					}
				}
			] )

			if( data.length ) cy.fit( undefined, 30 )

			// Add event listener for when node movement stops (mouseup after drag)
			cy.nodes().on( 'free', ( event: any ) => {
				const node = event.target
				const position = node.position()

				try {
					this.save_position( node.data('pathId'), position.x, position.y )
				} catch (err) {
					console.error( 'Failed to save position:', err )
				}
			} )
		}

		relayout() {
			const cy = this.cytoscape_instance()
			const { nodes, edges } = this.layout_input( false )

			const positions = graph_layout_positions( nodes, edges, {
				pin_saved: false,
				public_outside: this.public_outside(),
			} )

			cy.nodes().forEach( ( node: any ) => {
				const position = positions.get( node.id() )
				if( position ) node.position( position )
			} )
			cy.fit( undefined, 30 )

			this.save_message( 'Layout updated, positions not saved yet' )
		}

		save_all() {
			const cy = this.cytoscape_instance()
			const positions: { pathId: number, x: number, y: number }[] = []

			cy.nodes().forEach( ( node: any ) => {
				const position = node.position()
				positions.push( { pathId: node.data( 'pathId' ), x: position.x, y: position.y } )
			} )

			if( !positions.length ) return

			const res = mutate({
				input: { positions },
			})

			if( res.admin?.payload?.__typename === 'ErrorPayload' ) {
				this.save_message( 'Failed to save positions: ' + res.admin.payload.message )
				return
			}

			this.save_message( `Saved ${ positions.length } positions` )
		}

		save_position( pathId: string, x: number, y: number ) {
			const res = mutate({
				input: {
					positions: [{ pathId, x, y }]
				},
			})

			if( res.admin?.payload?.__typename === 'ErrorPayload' ) {
				console.error( 'Failed to save position:', res.admin.payload.message )
			}
		}
	}
}
