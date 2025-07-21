import { VSCodeDataGrid, VSCodeDataGridCell, VSCodeDataGridRow,  } from '@vscode/webview-ui-toolkit/react';
import { useState, useEffect } from 'react';
import { OcmResource } from '../../../src/data/loader'

export default function ShowPlacements() {
    let [placements, setPlacements] = useState<OcmResource[]>([]);

	useEffect(() => {
        window.addEventListener("message", event => {
			if ('crsDistribution' in event.data && 'Placement' === event.data.crsDistribution.kind) {
				setPlacements(JSON.parse(event.data.crsDistribution.crs));
			}
        });
    });

    return (
        <section className="component-row">
            { placements.length > 0 &&
                <>
                    <h2 style={{ marginTop: '40px' }}>Placements</h2>
                    <VSCodeDataGrid gridTemplateColumns="1fr 1fr 1fr 1fr" aria-label='Placement' >
                        <VSCodeDataGridRow rowType="sticky-header">
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='1'>Placement Name</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='2'>Namespace</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='3'>Number of Selected Clusters</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='4'>Conditions</VSCodeDataGridCell>
                        </VSCodeDataGridRow>

                        {placements.map(placement => {
                            return <VSCodeDataGridRow>
                                        <VSCodeDataGridCell gridColumn='1'>{placement.kr.metadata.name}</VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='2'>{placement.kr.metadata.namespace} </VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='3'>{placement.kr.status.numberOfSelectedClusters} </VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='4'>{placement.kr.status.conditions.map( ( condition:any )=> { return<p> - lastTransitionTime: {condition.lastTransitionTime}, message: {condition.message}, reason: {condition.reason}, status: {condition.status}, type: {condition.type} </p>  })} </VSCodeDataGridCell>
                                    </VSCodeDataGridRow>
                        } )
                        }
                    </VSCodeDataGrid>
                    <div style={{ borderTop: "1px solid #fff ", marginLeft: 10, marginRight: 10 }}></div>
                </>
            }
        </section>
    );
}
