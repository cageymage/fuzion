import { useLiveSurveyResults } from '../../hooks/useSurveyAdmin'
import { SurveyResultsView } from '../Survey/SurveyResults'
import styles from './OfficerSurveyManage.module.css'

export function LiveResults({ surveyId }: { surveyId: string }) {
  const results = useLiveSurveyResults(surveyId)

  return (
    <>
      <h2 className={styles.heading}>Live results</h2>
      <p className={styles.notice}>
        This survey is still open, so these results are provisional and refresh automatically.
      </p>
      {results.isPending && <p className={styles.notice}>Loading results…</p>}
      {results.isError && (
        <p role="alert" className={styles.error}>
          The results could not be loaded.
        </p>
      )}
      {results.isSuccess && (
        <>
          <SurveyResultsView results={results.data} />
          <p className={styles.notice}>Updated {new Date(results.dataUpdatedAt).toLocaleTimeString()}</p>
        </>
      )}
    </>
  )
}
