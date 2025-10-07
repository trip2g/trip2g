namespace $.$$ {
	export class $trip2g_admin_noteview_graph_cytoscape extends $.$trip2g_admin_noteview_graph_cytoscape {
		static cytoscape(): any {
			return $mol_import.script( 'https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.32.0/cytoscape.min.js' ).cytoscape
		}

		cytoscape(): any {
			return $trip2g_admin_noteview_graph_cytoscape.cytoscape()
		}

		@$mol_mem
		data() {
			const res = $trip2g_graphql_request( `
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

			return {
				nodes: res.admin.allLatestNoteViews.nodes,
				subgraphs: res.admin.allSubgraphs.nodes,
			}
		}

		@$mol_mem
		cytoscape_instance() {
			return this.cytoscape()( {
				container: this.dom_node()
			} )
		}

		render() {
			const cy = this.cytoscape_instance()
			const nodeFilter = ( item: any ) => !item.id.includes( 'sidebar' )
			const { nodes, subgraphs } = this.data()
			const data = nodes.filter( nodeFilter ).map( item => ( {
				...item,
				inLinks: item.inLinks.filter( nodeFilter ),
			} ) )

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

			// Prepare nodes and edges from the data
			const elements = []
			const nodeIds = new Set()

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

				if( node.graphPosition ) {
					element.position = node.graphPosition
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

			cy.layout({
				name: 'preset',

				// positions: (id) => {
				// 	console.log(id)
				// 	return {x:0, y:0}
				// },
			})

			// Apply layout
			// cy.layout( {
			// 	name: 'preset',
			// 	idealEdgeLength: 100,
			// 	nodeOverlap: 20,
			// 	refresh: 20,
			// 	fit: true,
			// 	padding: 30,
			// 	randomize: false,
			// 	componentSpacing: 100,
			// 	nodeRepulsion: 400000,
			// 	edgeElasticity: 100,
			// 	nestingFactor: 5,
			// 	gravity: 80,
			// 	numIter: 1000,
			// 	initialTemp: 200,
			// 	coolingFactor: 0.95,
			// 	minTemp: 1.0
			// } ).run()

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

		save_position( pathId: string, x: number, y: number ) {
			const res = $trip2g_graphql_request( `
					mutation AdminUpdateNoteGraphPositions($input: UpdateNoteGraphPositionsInput!) {
						admin {
							data: updateNoteGraphPositions(input: $input) {
								... on ErrorPayload {
									message
								}
								... on UpdateNoteGraphPositionsPayload {
									success
									updatedNoteViews {
										id
										pathId
										title
									}
								}
							}
						}
					}
				`, {
				input: { 
					positions: [{ pathId, x, y }]
				},
			} )

			if( res.admin?.data?.__typename === 'ErrorPayload' ) {
				console.error( 'Failed to save position:', res.admin.data.message )
			}
		}
	}
}
